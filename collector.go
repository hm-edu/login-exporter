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

	flag.StringVar(&logLevel, "log_level", "INFO", "Log level")

	flag.IntVar(&timeout, "timeout", 60, "Timeout in seconds")

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

// LoginStatus holds the result of a single login probe.
type LoginStatus struct {
	Success                 bool
	Elapsed                 float64
	ElapsedTotal            float64
	ElapsedLoginPageLoad    float64
	ElapsedLoginFormVisible float64
	ElapsedCredentials      float64
	ElapsedTotp             float64
}

// CaptureTime is a chromedp action that records the current time into the
// provided pointer when the action is executed.
type CaptureTime struct {
	time *time.Time
}

func (c CaptureTime) Do(ctx context.Context) error {
	*c.time = time.Now()
	return nil
}

// LogAction is a chromedp action that emits a structured log entry at Debug
// level when the action is executed.
type LogAction struct {
	target string
	part   string
	msg    string
}

func (l LogAction) Do(_ context.Context) error {
	logger.WithFields(log.Fields{
		"subsystem": "driver",
		"target":    l.target,
		"part":      l.part,
	}).Debug(l.msg)
	return nil
}

// getStatus Returns the data from the server
func getStatus(config SingleLoginConfig) LoginStatus {
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
		return LoginStatus{Success: false, Elapsed: -1, ElapsedTotal: -1, ElapsedLoginPageLoad: -1, ElapsedLoginFormVisible: -1, ElapsedCredentials: -1, ElapsedTotp: -1}
	}

	ctx, cancelTimeout := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancelTimeout()

	var text string
	var pageLoadTime time.Time
	var formVisibleTime time.Time
	var loginTime time.Time
	var credentialTime time.Time

	tasks := []chromedp.Action{
		LogAction{target: config.Target, part: "navigate", msg: "navigating to login URL"},
		chromedp.Navigate(config.Url),
		chromedp.WaitVisible(config.ExpectedHeaderCssClass),
		CaptureTime{time: &pageLoadTime},
		LogAction{target: config.Target, part: "page_load", msg: "login page loaded"},
		chromedp.Click(config.LoginCssClass, chromedp.NodeVisible),
		chromedp.WaitVisible(config.SubmitCssClass),
		CaptureTime{time: &formVisibleTime},
		LogAction{target: config.Target, part: "form_visible", msg: "login form is visible, submitting credentials"},
		chromedp.SendKeys(config.UsernameXpath, config.Username),
		chromedp.SendKeys(config.PasswordXpath, config.Password),
		chromedp.Click(config.SubmitCssClass),
	}

	if config.TotpSeed != "" {
		tasks = append(tasks,
			chromedp.WaitVisible(config.TotpXpath),
			CaptureTime{time: &credentialTime},
			LogAction{target: config.Target, part: "totp_prompt", msg: "credentials accepted, TOTP prompt visible"},
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
			LogAction{target: config.Target, part: "totp_submit", msg: "submitting TOTP code"},
			chromedp.Click(config.SubmitCssClass),
		)
	}

	tasks = append(tasks,
		chromedp.WaitVisible(config.ExpectedHeaderCssClass),
		chromedp.Text(config.ExpectedTextCssClass, &text),
		CaptureTime{time: &loginTime},
		LogAction{target: config.Target, part: "logged_in", msg: "login successful, navigating to logout URL"},
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
		return LoginStatus{Success: false, Elapsed: -1, ElapsedTotal: -1, ElapsedLoginPageLoad: -1, ElapsedLoginFormVisible: -1, ElapsedCredentials: -1, ElapsedTotp: -1}
	}

	if !strings.Contains(text, config.ExpectedText) {
		logger.WithFields(log.Fields{
			"subsystem": "driver",
			"target":    config.Target,
			"part":      "expected_text_check",
		}).Warningln("expected text not found in page, marking probe as failed")
		return LoginStatus{Success: false, Elapsed: -1, ElapsedTotal: -1, ElapsedLoginPageLoad: -1, ElapsedLoginFormVisible: -1, ElapsedCredentials: -1, ElapsedTotp: -1}
	}

	result := LoginStatus{
		Success:                 true,
		Elapsed:                 loginTime.Sub(start).Seconds(),
		ElapsedTotal:            stop.Sub(start).Seconds(),
		ElapsedLoginPageLoad:    pageLoadTime.Sub(start).Seconds(),
		ElapsedLoginFormVisible: formVisibleTime.Sub(start).Seconds(),
	}
	if config.TotpSeed != "" {
		result.ElapsedCredentials = credentialTime.Sub(start).Seconds()
		result.ElapsedTotp = loginTime.Sub(credentialTime).Seconds()
	} else {
		result.ElapsedCredentials = loginTime.Sub(start).Seconds()
		result.ElapsedTotp = -1
	}
	logger.WithFields(log.Fields{
		"subsystem":           "driver",
		"target":              config.Target,
		"part":                "probe_complete",
		"elapsed_page_load":   result.ElapsedLoginPageLoad,
		"elapsed_form":        result.ElapsedLoginFormVisible,
		"elapsed_credentials": result.ElapsedCredentials,
		"elapsed_totp":        result.ElapsedTotp,
		"elapsed_login":       result.Elapsed,
		"elapsed_total":       result.ElapsedTotal,
	}).Debug("probe completed successfully")
	return result
}

func init() {
	getCommandLineOptions()
	getLogger()
}
