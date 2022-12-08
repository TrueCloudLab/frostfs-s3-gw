package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/TrueCloudLab/frostfs-s3-gw/api"
	"github.com/TrueCloudLab/frostfs-s3-gw/authmate"
	"github.com/TrueCloudLab/frostfs-s3-gw/internal/frostfs"
	"github.com/TrueCloudLab/frostfs-s3-gw/internal/version"
	"github.com/TrueCloudLab/frostfs-s3-gw/internal/wallet"
	cid "github.com/TrueCloudLab/frostfs-sdk-go/container/id"
	"github.com/TrueCloudLab/frostfs-sdk-go/pool"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	poolDialTimeout        = 5 * time.Second
	poolHealthcheckTimeout = 5 * time.Second
	poolRebalanceInterval  = 30 * time.Second
	poolStreamTimeout      = 10 * time.Second

	// a month.
	defaultLifetime          = 30 * 24 * time.Hour
	defaultPresignedLifetime = 12 * time.Hour
)

type PoolConfig struct {
	Key                *ecdsa.PrivateKey
	Address            string
	DialTimeout        time.Duration
	HealthcheckTimeout time.Duration
	StreamTimeout      time.Duration
	RebalanceInterval  time.Duration
}

var (
	walletPathFlag           string
	accountAddressFlag       string
	peerAddressFlag          string
	eaclRulesFlag            string
	gateWalletPathFlag       string
	gateAccountAddressFlag   string
	accessKeyIDFlag          string
	containerIDFlag          string
	containerFriendlyName    string
	containerPlacementPolicy string
	gatesPublicKeysFlag      cli.StringSlice
	logEnabledFlag           bool
	logDebugEnabledFlag      bool
	sessionTokenFlag         string
	lifetimeFlag             time.Duration
	endpointFlag             string
	bucketFlag               string
	objectFlag               string
	methodFlag               string
	profileFlag              string
	regionFlag               string
	secretAccessKeyFlag      string
	containerPolicies        string
	awcCliCredFile           string
	timeoutFlag              time.Duration

	// pool timeouts flag.
	poolDialTimeoutFlag        time.Duration
	poolHealthcheckTimeoutFlag time.Duration
	poolRebalanceIntervalFlag  time.Duration
	poolStreamTimeoutFlag      time.Duration
)

const (
	envWalletPassphrase     = "wallet.passphrase"
	envWalletGatePassphrase = "wallet.gate.passphrase"
)

var zapConfig = zap.Config{
	Development: true,
	Encoding:    "console",
	Level:       zap.NewAtomicLevelAt(zapcore.FatalLevel),
	OutputPaths: []string{"stdout"},
	EncoderConfig: zapcore.EncoderConfig{
		MessageKey:   "message",
		LevelKey:     "level",
		EncodeLevel:  zapcore.CapitalLevelEncoder,
		TimeKey:      "time",
		EncodeTime:   zapcore.ISO8601TimeEncoder,
		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
	},
}

func prepare() (context.Context, *zap.Logger) {
	var (
		err    error
		log    = zap.NewNop()
		ctx, _ = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	)

	if !logEnabledFlag {
		return ctx, log
	} else if logDebugEnabledFlag {
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	if log, err = zapConfig.Build(); err != nil {
		panic(fmt.Errorf("create logger: %w", err))
	}

	return ctx, log
}

func main() {
	app := &cli.App{
		Name:     "FrostFS S3 Authmate",
		Usage:    "Helps manage delegated access via gates to data stored in FrostFS network",
		Version:  version.Version,
		Flags:    appFlags(),
		Commands: appCommands(),
	}
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("%s\nVersion: %s\nGoVersion: %s\n", c.App.Name, c.App.Version, runtime.Version())
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("AUTHMATE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AllowEmptyEnv(true)

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(100)
	}
}

func appFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "with-log",
			Usage:       "Enable logger",
			Destination: &logEnabledFlag,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Usage:       "Enable debug logger level",
			Destination: &logDebugEnabledFlag,
		},
		&cli.DurationFlag{
			Name: "timeout",
			Usage: "timeout of processing of the command, for example 2m " +
				"(note: max time unit is an hour so to set a day you should use 24h)",
			Destination: &timeoutFlag,
			Value:       1 * time.Minute,
		},
	}
}

func appCommands() []*cli.Command {
	return []*cli.Command{
		issueSecret(),
		obtainSecret(),
		generatePresignedURL(),
	}
}

func issueSecret() *cli.Command {
	return &cli.Command{
		Name:  "issue-secret",
		Usage: "Issue a secret in FrostFS network",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "wallet",
				Value:       "",
				Usage:       "path to the wallet",
				Required:    true,
				Destination: &walletPathFlag,
			},
			&cli.StringFlag{
				Name:        "address",
				Value:       "",
				Usage:       "address of wallet account",
				Required:    false,
				Destination: &accountAddressFlag,
			},
			&cli.StringFlag{
				Name:        "peer",
				Value:       "",
				Usage:       "address of a frostfs peer to connect to",
				Required:    true,
				Destination: &peerAddressFlag,
			},
			&cli.StringFlag{
				Name:        "bearer-rules",
				Usage:       "rules for bearer token (filepath or a plain json string are allowed)",
				Required:    false,
				Destination: &eaclRulesFlag,
			},
			&cli.StringSliceFlag{
				Name:        "gate-public-key",
				Usage:       "public 256r1 key of a gate (use flags repeatedly for multiple gates)",
				Required:    true,
				Destination: &gatesPublicKeysFlag,
			},
			&cli.StringFlag{
				Name:        "container-id",
				Usage:       "auth container id to put the secret into",
				Required:    false,
				Destination: &containerIDFlag,
			},
			&cli.StringFlag{
				Name:        "container-friendly-name",
				Usage:       "friendly name of auth container to put the secret into",
				Required:    false,
				Destination: &containerFriendlyName,
			},
			&cli.StringFlag{
				Name:        "container-placement-policy",
				Usage:       "placement policy of auth container to put the secret into",
				Required:    false,
				Destination: &containerPlacementPolicy,
				Value:       "REP 2 IN X CBF 3 SELECT 2 FROM * AS X",
			},
			&cli.StringFlag{
				Name:        "session-tokens",
				Usage:       "create session tokens with rules, if the rules are set as 'none', no session tokens will be created",
				Required:    false,
				Destination: &sessionTokenFlag,
				Value:       "",
			},
			&cli.DurationFlag{
				Name: "lifetime",
				Usage: `Lifetime of tokens. For example 50h30m (note: max time unit is an hour so to set a day you should use 24h). 
It will be ceil rounded to the nearest amount of epoch.`,
				Required:    false,
				Destination: &lifetimeFlag,
				Value:       defaultLifetime,
			},
			&cli.StringFlag{
				Name:        "container-policy",
				Usage:       "mapping AWS storage class to FrostFS storage policy as plain json string or path to json file",
				Required:    false,
				Destination: &containerPolicies,
			},
			&cli.StringFlag{
				Name:        "aws-cli-credentials",
				Usage:       "path to the aws cli credential file",
				Required:    false,
				Destination: &awcCliCredFile,
			},
			&cli.DurationFlag{
				Name:        "pool-dial-timeout",
				Usage:       `Timeout for connection to the node in pool to be established`,
				Required:    false,
				Destination: &poolDialTimeoutFlag,
				Value:       poolDialTimeout,
			},
			&cli.DurationFlag{
				Name:        "pool-healthcheck-timeout",
				Usage:       `Timeout for request to node to decide if it is alive`,
				Required:    false,
				Destination: &poolHealthcheckTimeoutFlag,
				Value:       poolHealthcheckTimeout,
			},
			&cli.DurationFlag{
				Name:        "pool-rebalance-interval",
				Usage:       `Interval for updating nodes health status`,
				Required:    false,
				Destination: &poolRebalanceIntervalFlag,
				Value:       poolRebalanceInterval,
			},
			&cli.DurationFlag{
				Name:        "pool-stream-timeout",
				Usage:       `Timeout for individual operation in streaming RPC`,
				Required:    false,
				Destination: &poolStreamTimeoutFlag,
				Value:       poolStreamTimeout,
			},
		},
		Action: func(c *cli.Context) error {
			ctx, log := prepare()

			password := wallet.GetPassword(viper.GetViper(), envWalletPassphrase)
			key, err := wallet.GetKeyFromPath(walletPathFlag, accountAddressFlag, password)
			if err != nil {
				return cli.Exit(fmt.Sprintf("failed to load frostfs private key: %s", err), 1)
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			poolCfg := PoolConfig{
				Key:                &key.PrivateKey,
				Address:            peerAddressFlag,
				DialTimeout:        poolDialTimeoutFlag,
				HealthcheckTimeout: poolHealthcheckTimeoutFlag,
				StreamTimeout:      poolStreamTimeoutFlag,
				RebalanceInterval:  poolRebalanceIntervalFlag,
			}

			frostFS, err := createFrostFS(ctx, log, poolCfg)
			if err != nil {
				return cli.Exit(fmt.Sprintf("failed to create FrostFS component: %s", err), 2)
			}

			agent := authmate.New(log, frostFS)

			var containerID cid.ID
			if len(containerIDFlag) > 0 {
				if err = containerID.DecodeString(containerIDFlag); err != nil {
					return cli.Exit(fmt.Sprintf("failed to parse auth container id: %s", err), 3)
				}
			}

			var gatesPublicKeys []*keys.PublicKey
			for _, key := range gatesPublicKeysFlag.Value() {
				gpk, err := keys.NewPublicKeyFromString(key)
				if err != nil {
					return cli.Exit(fmt.Sprintf("failed to load gate's public key: %s", err), 4)
				}
				gatesPublicKeys = append(gatesPublicKeys, gpk)
			}

			if lifetimeFlag <= 0 {
				return cli.Exit(fmt.Sprintf("lifetime must be greater 0, current value: %d", lifetimeFlag), 5)
			}

			policies, err := parsePolicies(containerPolicies)
			if err != nil {
				return cli.Exit(fmt.Sprintf("couldn't parse container policy: %s", err.Error()), 6)
			}

			bearerRules, err := getJSONRules(eaclRulesFlag)
			if err != nil {
				return cli.Exit(fmt.Sprintf("couldn't parse 'bearer-rules' flag: %s", err.Error()), 7)
			}

			sessionRules, skipSessionRules, err := getSessionRules(sessionTokenFlag)
			if err != nil {
				return cli.Exit(fmt.Sprintf("couldn't parse 'session-tokens' flag: %s", err.Error()), 8)
			}

			issueSecretOptions := &authmate.IssueSecretOptions{
				Container: authmate.ContainerOptions{
					ID:              containerID,
					FriendlyName:    containerFriendlyName,
					PlacementPolicy: containerPlacementPolicy,
				},
				FrostFSKey:            key,
				GatesPublicKeys:       gatesPublicKeys,
				EACLRules:             bearerRules,
				SessionTokenRules:     sessionRules,
				SkipSessionRules:      skipSessionRules,
				ContainerPolicies:     policies,
				Lifetime:              lifetimeFlag,
				AwsCliCredentialsFile: awcCliCredFile,
			}

			var tcancel context.CancelFunc
			ctx, tcancel = context.WithTimeout(ctx, timeoutFlag)
			defer tcancel()

			if err = agent.IssueSecret(ctx, os.Stdout, issueSecretOptions); err != nil {
				return cli.Exit(fmt.Sprintf("failed to issue secret: %s", err), 7)
			}
			return nil
		},
	}
}

func generatePresignedURL() *cli.Command {
	return &cli.Command{
		Name: "generate-presigned-url",
		Description: `Generate presigned url using AWS credentials. Credentials must be placed in ~/.aws/credentials.
You provide profile to load using --profile flag or explicitly provide credentials and region using
--aws-access-key-id, --aws-secret-access-key, --region.
Note to override credentials you must provide both access key and secret key.`,
		Usage: "generate-presigned-url --endpoint http://s3.frostfs.devenv:8080 --bucket bucket-name --object object-name --method get --profile aws-profile",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name: "lifetime",
				Usage: `Lifetime of presigned URL. For example 50h30m (note: max time unit is an hour so to set a day you should use 24h). 
It will be ceil rounded to the nearest amount of epoch.`,
				Required:    false,
				Destination: &lifetimeFlag,
				Value:       defaultPresignedLifetime,
			},
			&cli.StringFlag{
				Name:        "endpoint",
				Usage:       `Endpoint of s3-gw`,
				Required:    true,
				Destination: &endpointFlag,
			},
			&cli.StringFlag{
				Name:        "bucket",
				Usage:       `Bucket name to perform action`,
				Required:    true,
				Destination: &bucketFlag,
			},
			&cli.StringFlag{
				Name:        "object",
				Usage:       `Object name to perform action`,
				Required:    true,
				Destination: &objectFlag,
			},
			&cli.StringFlag{
				Name:        "method",
				Usage:       `HTTP method to perform action`,
				Required:    true,
				Destination: &methodFlag,
			},
			&cli.StringFlag{
				Name:        "profile",
				Usage:       `AWS profile to load`,
				Required:    false,
				Destination: &profileFlag,
			},
			&cli.StringFlag{
				Name:        "region",
				Usage:       `AWS region to use in signature (default is taken from ~/.aws/config)`,
				Required:    false,
				Destination: &regionFlag,
			},
			&cli.StringFlag{
				Name:        "aws-access-key-id",
				Usage:       `AWS access key id to sign the URL (default is taken from ~/.aws/credentials)`,
				Required:    false,
				Destination: &accessKeyIDFlag,
			},
			&cli.StringFlag{
				Name:        "aws-secret-access-key",
				Usage:       `AWS access secret access key to sign the URL (default is taken from ~/.aws/credentials)`,
				Required:    false,
				Destination: &secretAccessKeyFlag,
			},
		},
		Action: func(c *cli.Context) error {
			var cfg aws.Config
			if regionFlag != "" {
				cfg.Region = &regionFlag
			}
			if accessKeyIDFlag != "" && secretAccessKeyFlag != "" {
				cfg.Credentials = credentials.NewStaticCredentialsFromCreds(credentials.Value{
					AccessKeyID:     accessKeyIDFlag,
					SecretAccessKey: secretAccessKeyFlag,
				})
			}

			sess, err := session.NewSessionWithOptions(session.Options{
				Config:            cfg,
				Profile:           profileFlag,
				SharedConfigState: session.SharedConfigEnable,
			})
			if err != nil {
				return fmt.Errorf("couldn't get credentials: %w", err)
			}

			signer := v4.NewSigner(sess.Config.Credentials)
			req, err := http.NewRequest(strings.ToUpper(methodFlag), fmt.Sprintf("%s/%s/%s", endpointFlag, bucketFlag, objectFlag), nil)
			if err != nil {
				return fmt.Errorf("failed to create new request: %w", err)
			}

			date := time.Now().UTC()
			req.Header.Set(api.AmzDate, date.Format("20060102T150405Z"))

			if _, err = signer.Presign(req, nil, "s3", *sess.Config.Region, lifetimeFlag, date); err != nil {
				return fmt.Errorf("presign: %w", err)
			}

			res := &struct{ URL string }{
				URL: req.URL.String(),
			}

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.SetEscapeHTML(false)
			return enc.Encode(res)
		},
	}
}

func parsePolicies(val string) (authmate.ContainerPolicies, error) {
	if val == "" {
		return nil, nil
	}

	var (
		data = []byte(val)
		err  error
	)

	if !json.Valid(data) {
		if data, err = os.ReadFile(val); err != nil {
			return nil, fmt.Errorf("coudln't read json file or provided json is invalid")
		}
	}

	var policies authmate.ContainerPolicies
	if err = json.Unmarshal(data, &policies); err != nil {
		return nil, fmt.Errorf("unmarshal policies: %w", err)
	}
	if _, ok := policies[api.DefaultLocationConstraint]; ok {
		return nil, fmt.Errorf("config overrides %s location constraint", api.DefaultLocationConstraint)
	}

	return policies, nil
}

func getJSONRules(val string) ([]byte, error) {
	if val == "" {
		return nil, nil
	}
	data := []byte(val)
	if json.Valid(data) {
		return data, nil
	}

	if data, err := os.ReadFile(val); err == nil {
		if json.Valid(data) {
			return data, nil
		}
	}

	return nil, fmt.Errorf("coudln't read json file or provided json is invalid")
}

// getSessionRules reads json session rules.
// It returns true if rules must be skipped.
func getSessionRules(r string) ([]byte, bool, error) {
	if r == "none" {
		return nil, true, nil
	}

	data, err := getJSONRules(r)
	return data, false, err
}

func obtainSecret() *cli.Command {
	command := &cli.Command{
		Name:  "obtain-secret",
		Usage: "Obtain a secret from FrostFS network",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "wallet",
				Value:       "",
				Usage:       "path to the wallet",
				Required:    true,
				Destination: &walletPathFlag,
			},
			&cli.StringFlag{
				Name:        "address",
				Value:       "",
				Usage:       "address of wallet account",
				Required:    false,
				Destination: &accountAddressFlag,
			},
			&cli.StringFlag{
				Name:        "peer",
				Value:       "",
				Usage:       "address of frostfs peer to connect to",
				Required:    true,
				Destination: &peerAddressFlag,
			},
			&cli.StringFlag{
				Name:        "gate-wallet",
				Value:       "",
				Usage:       "path to the wallet",
				Required:    true,
				Destination: &gateWalletPathFlag,
			},
			&cli.StringFlag{
				Name:        "gate-address",
				Value:       "",
				Usage:       "address of wallet account",
				Required:    false,
				Destination: &gateAccountAddressFlag,
			},
			&cli.StringFlag{
				Name:        "access-key-id",
				Usage:       "access key id for s3",
				Required:    true,
				Destination: &accessKeyIDFlag,
			},
			&cli.DurationFlag{
				Name:        "pool-dial-timeout",
				Usage:       `Timeout for connection to the node in pool to be established`,
				Required:    false,
				Destination: &poolDialTimeoutFlag,
				Value:       poolDialTimeout,
			},
			&cli.DurationFlag{
				Name:        "pool-healthcheck-timeout",
				Usage:       `Timeout for request to node to decide if it is alive`,
				Required:    false,
				Destination: &poolHealthcheckTimeoutFlag,
				Value:       poolHealthcheckTimeout,
			},
			&cli.DurationFlag{
				Name:        "pool-rebalance-interval",
				Usage:       `Interval for updating nodes health status`,
				Required:    false,
				Destination: &poolRebalanceIntervalFlag,
				Value:       poolRebalanceInterval,
			},
			&cli.DurationFlag{
				Name:        "pool-stream-timeout",
				Usage:       `Timeout for individual operation in streaming RPC`,
				Required:    false,
				Destination: &poolStreamTimeoutFlag,
				Value:       poolStreamTimeout,
			},
		},
		Action: func(c *cli.Context) error {
			ctx, log := prepare()

			password := wallet.GetPassword(viper.GetViper(), envWalletPassphrase)
			key, err := wallet.GetKeyFromPath(walletPathFlag, accountAddressFlag, password)
			if err != nil {
				return cli.Exit(fmt.Sprintf("failed to load frostfs private key: %s", err), 1)
			}

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			poolCfg := PoolConfig{
				Key:                &key.PrivateKey,
				Address:            peerAddressFlag,
				DialTimeout:        poolDialTimeoutFlag,
				HealthcheckTimeout: poolHealthcheckTimeoutFlag,
				StreamTimeout:      poolStreamTimeoutFlag,
				RebalanceInterval:  poolRebalanceIntervalFlag,
			}

			frostFS, err := createFrostFS(ctx, log, poolCfg)
			if err != nil {
				return cli.Exit(fmt.Sprintf("failed to create FrostFS component: %s", err), 2)
			}

			agent := authmate.New(log, frostFS)

			var _ = agent

			password = wallet.GetPassword(viper.GetViper(), envWalletGatePassphrase)
			gateCreds, err := wallet.GetKeyFromPath(gateWalletPathFlag, gateAccountAddressFlag, password)
			if err != nil {
				return cli.Exit(fmt.Sprintf("failed to create owner's private key: %s", err), 4)
			}

			secretAddress := strings.Replace(accessKeyIDFlag, "0", "/", 1)

			obtainSecretOptions := &authmate.ObtainSecretOptions{
				SecretAddress:  secretAddress,
				GatePrivateKey: gateCreds,
			}

			var tcancel context.CancelFunc
			ctx, tcancel = context.WithTimeout(ctx, timeoutFlag)
			defer tcancel()

			if err = agent.ObtainSecret(ctx, os.Stdout, obtainSecretOptions); err != nil {
				return cli.Exit(fmt.Sprintf("failed to obtain secret: %s", err), 5)
			}

			return nil
		},
	}
	return command
}

func createFrostFS(ctx context.Context, log *zap.Logger, cfg PoolConfig) (authmate.FrostFS, error) {
	log.Debug("prepare connection pool")

	var prm pool.InitParameters
	prm.SetKey(cfg.Key)
	prm.SetNodeDialTimeout(cfg.DialTimeout)
	prm.SetHealthcheckTimeout(cfg.HealthcheckTimeout)
	prm.SetNodeStreamTimeout(cfg.StreamTimeout)
	prm.SetClientRebalanceInterval(cfg.RebalanceInterval)
	prm.AddNode(pool.NewNodeParam(1, cfg.Address, 1))

	p, err := pool.NewPool(prm)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err = p.Dial(ctx); err != nil {
		return nil, fmt.Errorf("dial pool: %w", err)
	}

	return frostfs.NewAuthmateFrostFS(p), nil
}
