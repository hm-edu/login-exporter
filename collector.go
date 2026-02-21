package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/pquerna/otp/totp"
	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v3"
)

// LoginConfigs Is the list of configuration
// That should be read
type LoginConfigs struct {
	Configs []SingleLoginConfig `yaml:"targets"`
}

// SingleLoginConfig Is the login configuration settings
// that is used to read the yaml files.
type SingleLoginConfig struct {
	Url                    string `yaml:"url"`
	Target                 string `yaml:"target"`
	ExpectedHeaderCssClass string `yaml:"expected_header_css_class"`
	ExpectedTextCssClass   string `yaml:"expected_text_css_class"`
	LoginCssClass          string `yaml:"login_css_class"`
	Username               string `yaml:"username"`
	Password               string `yaml:"password"`
	TotpSeed               string `yaml:"totp_seed"`
	UsernameXpath          string `yaml:"username_xpath"`
	PasswordXpath          string `yaml:"password_xpath"`
	TotpXpath              string `yaml:"totp_xpath"`
	SubmitCssClass         string `yaml:"submit_css_class"`
	ExpectedText           string `yaml:"expected_text"`
	LoginType              string `yaml:"login_type"`
	LogoutUrl              string `yaml:"logout_url"`
}

// readConfig Reads the yaml configuration from the given server
func readConfig(path string) LoginConfigs {
	var loginConfigs LoginConfigs
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		logger.WithFields(
			log.Fields{
				"subsystem": "config_loader",
				"part":      "read_file",
			}).Panicln(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &loginConfigs)
	if err != nil {
		logger.WithFields(
			log.Fields{
				"subsystem": "config_loader",
				"part":      "parse_file",
			}).Panicln(err.Error())
	}
	return loginConfigs
}

// Configuration Params
var configFilePath string
var listenIp string
var listenPort int

// Logging Params
var logPath string
var logLevel string

// Timeout settings
var timeout int

var logger = log.New()

// getCommandLineOptions Returns the command options from the terminal
func getCommandLineOptions() {
	flag.StringVar(&configFilePath, "config", "/etc/prometheus/login.yml", "Configuration file path")
	flag.StringVar(&listenIp, "listen_ip", "127.0.0.1", "Listen IP Address")
	flag.IntVar(&listenPort, "listen_port", 9980, "Listen Port")

	flag.StringVar(&logPath, "log_file", "login_exporter.log", "Log file path")
	flag.StringVar(&logLevel, "log_level", "INFO", "Log level")

	flag.IntVar(&timeout, "timeout", 120, "Timeout in seconds")

	flag.Parse()
}

// getLogger Returns the logger that is used to log the data
func getLogger() *log.Logger {
	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	parsedLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	logger.SetLevel(parsedLevel)
	return logger
}

type LoginResult struct {
	time *time.Time
}

func (l LoginResult) Do(ctx context.Context) error {
	*l.time = time.Now()
	return nil
}

// getStatus Returns the data from the server
func getStatus(config SingleLoginConfig) (status bool, elapsed float64, elapsedTotal float64) {
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.DisableGPU)...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancelCtx()

	// warmup the browser
	if err := chromedp.Run(ctx); err != nil {
		logger.WithFields(
			log.Fields{
				"subsystem": "driver",
				"part":      "warmup",
			}).Warningln(err.Error())
		return false, -1, -1
	}

	ctx, cancelTimeout := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancelTimeout()

	var text string
	var loginTime time.Time

	tasks := []chromedp.Action{
		chromedp.Navigate(config.Url),
		chromedp.WaitVisible(config.ExpectedHeaderCssClass),
		chromedp.Click(config.LoginCssClass, chromedp.NodeVisible),
		chromedp.WaitVisible(config.SubmitCssClass),
		chromedp.SendKeys(config.UsernameXpath, config.Username),
		chromedp.SendKeys(config.PasswordXpath, config.Password),
		chromedp.Click(config.SubmitCssClass),
	}

	if config.TotpSeed != "" {
		tasks = append(tasks,
			chromedp.WaitVisible(config.TotpXpath),
			chromedp.QueryAfter(config.TotpXpath, func(ctx context.Context, execCtx runtime.ExecutionContextID, nodes ...*cdp.Node) error {
				if len(nodes) < 1 {
					return fmt.Errorf("selector %q did not return any nodes", config.TotpXpath)
				}
				otp, err := totp.GenerateCode(config.TotpSeed, time.Now())
				if err != nil {
					return fmt.Errorf("failed to generate OTP: %v", err)
				}
				return chromedp.KeyEventNode(nodes[0], otp).Do(ctx)
			}, chromedp.NodeVisible),
			chromedp.Click(config.SubmitCssClass),
		)
	}

	tasks = append(tasks,
		chromedp.WaitVisible(config.ExpectedHeaderCssClass),
		chromedp.Text(config.ExpectedTextCssClass, &text),
		LoginResult{time: &loginTime},
		chromedp.Navigate(config.LogoutUrl),
	)

	start := time.Now()
	err := chromedp.Run(ctx, tasks...)
	stop := time.Now()

	if err != nil {
		logger.WithFields(
			log.Fields{
				"subsystem": "driver",
				"part":      "navigation_error",
			}).Warningln(err.Error())
		return false, -1, -1
	}

	if strings.Contains(text, config.ExpectedText) {
		status = true
		elapsed = loginTime.Sub(start).Seconds()
		elapsedTotal = stop.Sub(start).Seconds()
	} else {
		elapsed = -1
		elapsedTotal = -1
	}

	return status, elapsed, elapsedTotal
}

func init() {
	getCommandLineOptions()
	getLogger()
}
