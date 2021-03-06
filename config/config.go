package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/viper"

	flag "github.com/spf13/pflag"
)

// DefaultKubeConfigLocation is the default location of the KubeConfig file.
const DefaultKubeConfigLocation = "/.kube/config"

// Configuration contains all the User Parameter for Trireme-Kubernetes.
type Configuration struct {
	// AuthType defines if Trireme uses PSK or PKI
	AuthType string
	// KubeNodeName is the identifier used for this Trireme instance
	KubeNodeName string
	// PKIDirectory is the directory where the Key files are stored (is using PKI)
	PKIDirectory string
	// TriremePSK is the PSK used for Trireme (if using PSK)
	TriremePSK string
	// RemoteEnforcer defines if the enforcer is spawned into each POD namespace
	// or into the host default namespace.
	RemoteEnforcer bool

	TriremeNetworks       string
	ParsedTriremeNetworks []string

	KubeconfigPath string
	LogLevel       string

	// Enforce defines if this process is an enforcer process (spawned into POD namespaces)
	Enforce bool `mapstructure:"Enforce"`
}

func usage() {
	flag.PrintDefaults()
	os.Exit(2)
}

// LoadConfig loads a Configuration struct:
// 1) If presents flags are used
// 2) If no flags, Env Variables are used
// 3) If no Env Variables, defaults are used when possible.
func LoadConfig() (*Configuration, error) {
	flag.Usage = usage
	flag.String("AuthType", "", "Authentication type: PKI/PSK")
	flag.String("KubeNodeName", "", "Node name in Kubernetes")
	flag.String("PKIDirectory", "", "Directory where the Trireme PKIs are")
	flag.String("TriremePSK", "", "PSK to use")
	flag.Bool("RemoteEnforcer", true, "Use the Trireme Remote Enforcer.")
	flag.String("TriremeNetworks", "", "TriremeNetworks")
	flag.String("KubeconfigPath", "", "KubeConfig used to connect to Kubernetes")
	flag.String("LogLevel", "", "Log level. Default to info (trace//debug//info//warn//error//fatal)")
	flag.Bool("Enforce", false, "Run Trireme-Kubernetes in Enforce mode.")

	// Setting up default configuration
	viper.SetDefault("AuthType", "PSK")
	viper.SetDefault("KubeNodeName", "")
	viper.SetDefault("PKIDirectory", "")
	viper.SetDefault("TriremePSK", "PSK")
	viper.SetDefault("RemoteEnforcer", true)
	viper.SetDefault("TriremeNetworks", "")
	viper.SetDefault("KubeconfigPath", "")
	viper.SetDefault("LogLevel", "info")
	viper.SetDefault("Enforce", false)

	// Binding ENV variables
	// Each config will be of format TRIREME_XYZ as env variable, where XYZ
	// is the upper case config.
	viper.SetEnvPrefix("TRIREME")
	viper.AutomaticEnv()

	// Binding CLI flags.
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)

	var config Configuration

	// Manual check for Enforce mode as this is given as a simple argument
	if len(os.Args) > 1 {
		if os.Args[1] == "enforce" {
			config.Enforce = true
			return &config, nil
		}
	}

	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling:%s", err)
	}

	err = validateConfig(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// validateConfig is validating the Configuration struct.
func validateConfig(config *Configuration) error {
	// Validating KUBECONFIG
	// In case not running as InCluster, we try to infer a possible KubeConfig location
	if os.Getenv("KUBERNETES_PORT") == "" {
		if config.KubeconfigPath == "" {
			config.KubeconfigPath = os.Getenv("HOME") + DefaultKubeConfigLocation
		}
	} else {
		config.KubeconfigPath = ""
	}

	// Validating KUBE NODENAME
	if !config.Enforce && config.KubeNodeName == "" {
		return fmt.Errorf("Couldn't load NodeName. Ensure Kubernetes Nodename is given as a parameter")
	}

	// Validating AUTHTYPE
	if config.AuthType != "PSK" && config.AuthType != "PKI" {
		return fmt.Errorf("AuthType should be PSK or PKI")
	}

	// Validating PSK
	if config.AuthType == "PSK" && config.TriremePSK == "" {
		return fmt.Errorf("PSK should be provided")
	}

	parsedTriremeNetworks, err := parseTriremeNets(config.TriremeNetworks)
	if err != nil {
		return fmt.Errorf("TargetNetwork is invalid: %s", err)
	}
	config.ParsedTriremeNetworks = parsedTriremeNetworks

	return nil
}

// parseTriremeNets
func parseTriremeNets(nets string) ([]string, error) {
	resultNets := strings.Fields(nets)

	// Validation of each networks.
	for _, network := range resultNets {
		_, _, err := net.ParseCIDR(network)
		if err != nil {
			return nil, fmt.Errorf("Invalid CIDR: %s", err)
		}
	}
	return resultNets, nil
}
