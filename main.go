package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/charmbracelet/lipgloss"
	"github.com/cli/cli/v2/pkg/surveyext"
	"github.com/google/go-jsonnet"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/passbolt/go-passbolt/api"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	ky "sigs.k8s.io/yaml"

	"github.com/vshn/emergency-credentials-receive/pkg/config"
	"github.com/vshn/emergency-credentials-receive/pkg/inputs"
)

//go:embed kubectl-config-tmpl.jsonnet
var kubectlTemplate string

var sampleConfig = `
	passbolt_key: |
		-----BEGIN PGP PRIVATE KEY BLOCK-----
		Version: OpenPGP.js v4.10.9
		Comment: https://openpgpjs.org

		[...]

		-----END PGP PRIVATE KEY BLOCK-----
`

const (
	envVarPassphrase         = "EMR_PASSPHRASE"
	envVarKubernetesEndpoint = "EMR_KUBERNETES_ENDPOINT"

	defaultEndpoint           = "https://cloud.passbolt.com/vshn"
	defaultKubernetesEndpoint = "https://kubernetes.default.svc:6443"
	// defaultEmergencyCredentialsBucketConfigName is the name of the resource in passbolt that contains the bucket configuration.
	defaultEmergencyCredentialsBucketConfigName = "emergency-cedentials-buckets"

	clusterOverviewPage = "https://wiki.vshn.net/x/4whJF"

	userAgent = "emergency-credentials-receive/0.0.0"
)

var (
	tokenOutputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	boldStyle        = lipgloss.NewStyle().Bold(true)
	isTerminal       = term.IsTerminal(int(os.Stdout.Fd()))

	omitTokenOutput bool
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] [cluster_id]\n\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Available env variables:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\t%s\t\tthe passphrase to unlock the Passbolt key\n", envVarPassphrase)
		fmt.Fprintf(flag.CommandLine.Output(), "\t%s\tthe Kubernetes endpoint written to the created kubeconfig file\n", envVarKubernetesEndpoint)
		fmt.Fprintf(flag.CommandLine.Output(), "\t%s\t\tthe directory the configuration is stored in\n", config.EnvConfigDir)

		flag.PrintDefaults()
	}
	flag.BoolVar(&omitTokenOutput, "omit-token-output", false, "omit token output to STDOUT")
	flag.Parse()

	clusterId := flag.Arg(0)

	var saveConfig bool
	lln("Welcome to the Emergency Credentials Receive tool!")
	lf("This tool will help you receive your cluster emergency credentials from Passbolt.\n\n")

	c, err := config.RetrieveConfig()
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		lf("No config file found at %q.\n", config.ConfigFile())
		lln("File will be created after a successful login.")
	} else if err != nil {
		lln("Error retrieving config: ", err)
	}

	if c.PassboltKey == "" && isTerminal {
		k, err := surveyext.Edit("", "", "\n\n# Paste your Passbolt private key from\n#  https://cloud.passbolt.com/vshn/app/settings/keys\n", os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			lln("Error retrieving passbolt key: ", err)
			os.Exit(1)
		}
		saveConfig = true
		c.PassboltKey = k
	}
	if c.PassboltKey == "" {
		lf("Passbolt key cannot be empty. Please provide interactively or create a config file at %q:\n%s", config.ConfigFile(), sampleConfig)
		os.Exit(1)
	}

	passphrase := os.Getenv("EMR_PASSPHRASE")
	if passphrase == "" && isTerminal {
		pf, err := inputs.PassphraseInput("Enter your Passbolt passphrase", "")
		if err != nil {
			lln("Error retrieving passbolt passphrase: ", err)
			os.Exit(1)
		}
		passphrase = pf
	} else if passphrase == "" {
		lln("Passphrase cannot be empty.")
		lln("Provide interactively or set EMR_PASSPHRASE environment variable.")
		os.Exit(1)
	} else {
		lln("Using passphrase from EMR_PASSPHRASE environment variable.")
	}

	if clusterId == "" {
		cid, err := inputs.LineInput("Enter the ID of the cluster you want to access", "c-crashy-wreck-1234")
		if err != nil {
			lln("Error retrieving cluster ID: ", err)
			os.Exit(1)
		}
		clusterId = cid
	}
	if clusterId == "" {
		lln("Cluster ID cannot be empty.")
		lln("Provide interactively or as argument.")
		os.Exit(1)
	}

	client, err := api.NewClient(nil, userAgent, defaultEndpoint, c.PassboltKey, passphrase)
	if err != nil {
		lf("Error creating passbolt client: %v\n", err)
		os.Exit(1)
	}

	lln("Logging into passbolt...")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := client.Login(ctx); err != nil {
		lf("Error logging into passbolt: %v\n", err)
		os.Exit(1)
	}

	if saveConfig {
		if err := config.SaveConfig(c); err != nil {
			lln("Error saving config: ", err)
		}
	}

	lf("Logged in. Retrieving bucket configuration from %q...\n", defaultEmergencyCredentialsBucketConfigName)
	res, err := client.GetResources(ctx, &api.GetResourcesOptions{})
	if err != nil {
		lln("Error retrieving resources from passbolt: ", err)
	}

	var resource api.Resource
	for _, r := range res {
		if r.Name == defaultEmergencyCredentialsBucketConfigName {
			resource = r
			break
		}
	}
	if resource.ID == "" {
		lln("Error retrieving bucket configuration from passbolt: ", fmt.Errorf("could not find resource %q", defaultEmergencyCredentialsBucketConfigName))
		os.Exit(1)
	}
	lln("  Retrieving bucket secret...")
	secret, err := client.GetSecret(ctx, resource.ID)
	if err != nil {
		lln("Error retrieving bucket secret from passbolt: ", err)
		os.Exit(1)
	}

	lln("  Decrypting bucket secret...")
	conf, err := client.DecryptMessage(secret.Data)
	if err != nil {
		lln("Error decrypting bucket secret in passbolt: ", err)
		os.Exit(1)
	}

	lln("  Parsing passbolt secret...")
	var pbsc api.SecretDataTypePasswordAndDescription
	if err := json.Unmarshal([]byte(conf), &pbsc); err != nil {
		lln("Error parsing the decrypted passbolt secret: ", err)
		os.Exit(1)
	}
	lln("  Parsing bucket configuration from secret...")
	var bc bucketConfig
	if err := yaml.Unmarshal([]byte(pbsc.Password), &bc); err != nil {
		lln("Error parsing bucket configuration from passbolt secrets password field: ", err)
		os.Exit(1)
	}

	lf("%d buckets with credentials found\n", len(bc.Buckets))

	var emcreds []string
	for _, b := range bc.Buckets {
		lf("Trying %q (bucket %q, region %q, keyId %q)\n", b.Endpoint, b.Bucket, b.Region, b.AccessKeyId)

		mc, err := minio.New(b.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(b.AccessKeyId, b.SecretAccessKey, ""),
			Secure: !b.Insecure,
			Region: b.Region,
		})
		if err != nil {
			lln("  Error creating minio client: ", err)
			continue
		}

		objectName := clusterId
		if b.ObjectNameTemplate != "" {
			lln("  Constructing object name from template...")
			t, err := template.New("fileName").Funcs(sprig.TxtFuncMap()).Parse(b.ObjectNameTemplate)
			if err != nil {
				lln("    unable to parse file name template:", err)
				continue
			}
			buf := new(strings.Builder)
			if err := t.Execute(buf, struct {
				ClusterId string
				Context   map[string]string
			}{
				ClusterId: clusterId,
				Context:   map[string]string{"ClusterId": clusterId},
			}); err != nil {
				lln("    unable to execute file name template:", err)
				continue
			}
			objectName = buf.String()
		}
		lf("  Downloading %q...\n", objectName)

		// fully read object into memory, otherwise the error message can be very confusing
		// it says something like "JSON unmarshal error: not found"
		buf, err := minioGetReadAll(ctx, mc, b.Bucket, objectName)
		if err != nil {
			lln("    Error downloading object: ", err)
			continue
		}

		lln("  Parsing object...")
		var et encryptedToken
		if err := json.Unmarshal(buf, &et); err != nil {
			lln("    Error parsing object: ", err)
			continue
		}

		lln("  Trying to decrypt object...")
		var decrypted string
		for _, s := range et.Secrets {
			d, err := client.DecryptMessage(s.Data)
			if err == nil {
				decrypted = d
				break
			}
		}
		if decrypted == "" {
			lln("    No decryptable secret found")
			continue
		}

		emcreds = append(emcreds, decrypted)
	}

	if len(emcreds) == 0 {
		lln("No valid emergency credentials found")
		os.Exit(1)
	}

	lf("Emergency credentials found\n\n")
	for i, c := range emcreds {
		fmt.Println("# ", "Token", i)
		if omitTokenOutput {
			fmt.Println(tokenOutputStyle.Render("*** OMITTED ***"))
		} else {
			fmt.Println(tokenOutputStyle.Render(c))
		}
	}

	kep := os.Getenv("EMR_KUBERNETES_ENDPOINT")
	if kep == "" && isTerminal {
		ih := fmt.Sprintf("Provide API endpoint to render kubeconfig. See %q for an overview.", clusterOverviewPage)
		k, err := inputs.LineInput(ih, defaultKubernetesEndpoint)
		if err != nil {
			lln("Error retrieving kubernetes endpoint: ", err)
			os.Exit(1)
		}
		kep = k
	}
	if kep == "" {
		lln("Assuming default kubernetes endpoint.")
		kep = defaultKubernetesEndpoint
	}
	kubeconfig, err := renderKubeconfig(kep, emcreds)
	if err != nil {
		lln("Error rendering kubeconfig: ", err)
		lln("The tokens printed above should continue to work, but you will have to create the kubeconfig manually.")
		os.Exit(1)
	}

	kcFileName := "em-" + clusterId
	if err := os.WriteFile(kcFileName, []byte("# Generated by emergency-credentials-receive\n"+kubeconfig), 0600); err != nil {
		lln("Error writing kubeconfig: ", err)
		lln("The tokens printed above should continue to work, but you will have to create the kubeconfig manually.")
		os.Exit(1)
	}
	lf("Wrote kubeconfig to %q. Use with:\n\n", kcFileName)
	lln(boldStyle.Render(fmt.Sprintf("export KUBECONFIG=%q", kcFileName)))
	lln(boldStyle.Render("kubectl get nodes"))
}

// minioGetReadAll is a helper function to fully download a S3 object into memory.
func minioGetReadAll(ctx context.Context, mc *minio.Client, bucket, objectName string) ([]byte, error) {
	object, err := mc.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer object.Close()

	return io.ReadAll(object)
}

func lf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func lln(a ...any) {
	fmt.Fprintln(os.Stderr, a...)
}

func renderKubeconfig(server string, tokens []string) (string, error) {
	vm := jsonnet.MakeVM()

	tks, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}

	vm.ExtVar("server", server)
	vm.ExtCode("tokens", string(tks))

	json, err := vm.EvaluateAnonymousSnippet("kubeconfig", kubectlTemplate)
	if err != nil {
		return "", err
	}

	yml, err := ky.JSONToYAML([]byte(json))
	return string(yml), err
}

type bucketConfig struct {
	Buckets []bucket `yaml:"buckets"`
}

type bucket struct {
	// Endpoint is the S3 endpoint to use.
	Endpoint string `yaml:"endpoint"`
	// Bucket is the S3 bucket to use.
	Bucket string `yaml:"bucket"`

	// AccessKeyId and SecretAccessKey are the S3 credentials to use.
	AccessKeyId string `yaml:"accessKeyId"`
	// SecretAccessKey is the S3 secret access key to use.
	SecretAccessKey string `yaml:"secretAccessKey"`

	// Region is the AWS region to use.
	Region string `yaml:"region,omitempty"`
	// Insecure allows to use an insecure connection to the S3 endpoint.
	Insecure bool `yaml:"insecure,omitempty"`

	// ObjectNameTemplate is a template for the object name to use.
	ObjectNameTemplate string `yaml:"objectNameTemplate,omitempty"`
}

// encryptedToken is the JSON structure of an encrypted token.
type encryptedToken struct {
	Secrets []encryptedTokenSecret `json:"secrets"`
}

// encryptedTokenSecret is the JSON structure of an encrypted token secret.
type encryptedTokenSecret struct {
	Data string `json:"data"`
}
