package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
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
	Url               string `yaml:"url"`
	Target            string `yaml:"target"`
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	Certificate       string `yaml:"certificate"`
	UsernameXpath     string `yaml:"username_xpath"`
	PasswordXpath     string `yaml:"password_xpath"`
	CertificateXpath  string `yaml:"certificate_xpath"`
	SubmitXpath       string `yaml:"submit_xpath"`
	LoginType         string `yaml:"login_type"`
	ExpectedText      string `yaml:"expected_text"`
	ExpectedTextXpath string `yaml:"expected_text_xpath"`
	ExpectedTextFrame string `yaml:"expected_text_frame"`
	SSLCheck          bool   `yaml:"ssl_check"`
	Debug             bool   `yaml:"debug"`
	Method            string `yaml:"method"`
	SubmitType        string `yaml:"submit_type"`
	LogoutXpath       string `yaml:"logout_xpath"`
	LogoutSubmitType  string `yaml:"logout_submit_type"`
	LogoutFrame       string `yaml:"logout_frame"`
	LogoutUrl         string `yaml:"logout_url"`
	WaitTime          int    `yaml:"wait_time"`
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

	flag.IntVar(&timeout, "timeout", 60, "Timeout in seconds")

	flag.Parse()
}

// getLogger Returns the logger that is used to log the data
func getLogger() *log.Logger {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0660)
	if err != nil {
		panic(err)
	}
	logger.SetFormatter(&log.JSONFormatter{})
	logger.SetOutput(f)
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

func (l LoginResult) Do(context.Context) error {
	*l.time = time.Now()
	return nil
}

// loginShibboleth Logs in the shibboleth system using the given username and password
func loginShibboleth(urlText string, username string, password string, usernameXpath string,
	passwordXpath string, submitXpath string, logoutUrl string, text *string, loginTime *time.Time) []chromedp.Action {
	actions := []chromedp.Action{}
	if usernameXpath == "" && passwordXpath == "" && submitXpath == "" {
		usernameXpath = "//input[@id='username']"
		passwordXpath = "//input[@id='password']"
		submitXpath = "//button[@class='aai_login_button']"
	}
	actions = append(actions,
		chromedp.WaitVisible(usernameXpath), chromedp.SendKeys(usernameXpath, username), chromedp.SendKeys(passwordXpath, password),
		chromedp.Click(submitXpath), chromedp.WaitVisible("//pre"), chromedp.Text("body", text), LoginResult{time: loginTime}, chromedp.Navigate(logoutUrl), chromedp.WaitVisible("//*[@id='propagate_yes']"), chromedp.Click("//*[@id='propagate_yes']"), chromedp.WaitVisible("//*[@class='logout success']"))
	return actions
}

// getStatus Returns the data from the server
func getStatus(config SingleLoginConfig) (status bool, elapsed float64, elapsedTotal float64) {

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// ensure that the browser process is started
	if err := chromedp.Run(taskCtx); err != nil {
		log.Fatal(err)
	}

	status = false
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var text string
	var loginTime time.Time
	tasks := []chromedp.Action{chromedp.Navigate(config.Url)}
	tasks = append(tasks, loginShibboleth(config.Url, config.Username, config.Password, config.UsernameXpath, config.PasswordXpath, config.SubmitXpath, config.LogoutUrl, &text, &loginTime)...)

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	start := time.Now()

	err := chromedp.Run(ctx,
		tasks...,
	)

	stop := time.Now()
	if err != nil {
		logger.WithFields(
			log.Fields{
				"subsystem": "driver",
				"part":      "navigation_error",
			}).Warningln(err.Error())
	}
	if strings.Contains(text, config.ExpectedText) {
		status = true
	} else {
		status = false
	}

	elapsed = loginTime.Sub(start).Seconds()
	elapsedTotal = stop.Sub(start).Seconds()
	return status, elapsed, elapsedTotal
}

func init() {
	getCommandLineOptions()
	getLogger()
}
