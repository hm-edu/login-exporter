# Login Exporter

Login Exporter is a Prometheus exporter that uses [chromedp](https://github.com/chromedp/chromedp)
to drive a headless Chrome browser, log in to a web application, and verify that an expected
text is present in the result. It simulates real end-user login interactions and can be used
for availability monitoring and status management of web applications.

## Installation

The exporter communicates with Chrome directly via the Chrome DevTools Protocol (CDP) using
`chromedp`. No separate ChromeDriver binary is required — you only need
Chrome or Chromium installed on the machine.

### Install on macOS

```bash
brew install --cask google-chrome
```

### Install on Ubuntu

```bash
# Install Chrome.
sudo curl -sS -o - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add
sudo echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list
sudo apt-get -y update
sudo apt-get -y install google-chrome-stable
```

## Configuration

The following command-line parameters are available:

| parameter      | default                     | description |
|----------------|-----------------------------|-------------|
| `-config`      | `/etc/prometheus/login.yml` | Path to the login configuration file (must be readable by the running user) |
| `-listen_ip`   | `127.0.0.1`                 | IP address the exporter listens on |
| `-listen_port` | `9980`                      | Port the exporter listens on |
| `-log_level`   | `INFO`                      | Log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `-timeout`     | `60`                        | Timeout in seconds before a probe is aborted |

Logs are written to **stdout** in JSON format.

## login.yml

The configuration file defines a list of `targets`. Each target describes one login probe.
An example file is available at `misc/login.yml.dist`.

### Configuration fields

| field                       | required | description |
|-----------------------------|----------|-------------|
| `target`                    | yes      | Unique name used to look up this config and as a Prometheus label |
| `url`                       | yes      | URL of the page containing the login entry point |
| `login_type`                | yes      | Arbitrary string used as a `login_type` label on all metrics (e.g. `shibboleth`) |
| `expected_header_css_class` | yes      | CSS selector that must be visible after the page loads (signals page-load completion) |
| `login_css_class`           | yes      | CSS selector for the button/element that opens the login form |
| `username_xpath`            | yes      | XPath of the username input field |
| `password_xpath`            | yes      | XPath of the password input field |
| `submit_css_class`          | yes      | CSS selector for the login submit button |
| `username`                  | yes      | Username to submit |
| `password`                  | yes      | Password to submit |
| `expected_text_css_class`   | yes      | CSS selector of the element whose text content is checked after login |
| `expected_text`             | yes      | Text that must appear in the element matched by `expected_text_css_class` |
| `logout_url`                | yes      | URL to navigate to after the check (performs logout) |
| `totp_seed`                 | no       | Base-32 TOTP seed; when set, a TOTP step is performed after credentials |
| `totp_xpath`                | no       | XPath of the TOTP input field (required when `totp_seed` is set) |

### Login flow

1. Navigate to `url`.
2. Wait for `expected_header_css_class` to become visible (page-load milestone).
3. Click `login_css_class` to open the login form.
4. Wait for `submit_css_class` to become visible (form-visible milestone).
5. Enter `username` into `username_xpath` and `password` into `password_xpath`.
6. Click `submit_css_class`.
7. *(Optional, when `totp_seed` is set)* Wait for `totp_xpath`, generate a TOTP code from
   `totp_seed`, enter the code, and click `submit_css_class` again.
8. Wait for `expected_header_css_class` to become visible again (logged-in milestone).
9. Read the text content of `expected_text_css_class` and verify it contains `expected_text`.
10. Navigate to `logout_url`.

### Example

```yaml
targets:
  - url: "https://example.com"
    target: "my-app"
    login_type: "shibboleth"
    expected_header_css_class: "h2.PageTitle"
    expected_text_css_class: "h5.WelcomeMessage"
    login_css_class: "button.LoginButton"
    username_xpath: "//input[@id='username']"
    password_xpath: "//input[@id='password']"
    submit_css_class: "button.form-button"
    username: "myuser"
    password: "mypassword"
    totp_seed: "BASE32TOTPSEEDHERE"
    totp_xpath: "//input[@id='otp_input']"
    expected_text: "Welcome, myuser"
    logout_url: "https://sso.example.com/idp/profile/Logout"
```

## Exposed Metrics

All metrics carry `target` and `login_type` labels.

| metric                               | description |
|--------------------------------------|-------------|
| `login_status`                       | `1` for success, `0` for failure |
| `login_elapsed_seconds`              | Total time from start until the logged-in state was confirmed |
| `login_total_elapsed_seconds`        | Total time including logout navigation |
| `login_page_load_elapsed_seconds`    | Time until the login page finished loading |
| `login_form_visible_elapsed_seconds` | Time until the login form became visible |
| `login_credentials_elapsed_seconds`  | Time for the credential step (excludes TOTP when TOTP is used) |
| `login_totp_elapsed_seconds`         | Time for the TOTP step (`-1` when TOTP is not used) |

On any error all timing metrics are set to `-1`.

## Configuring Prometheus

This exporter works like the
[blackbox exporter](https://github.com/prometheus/blackbox_exporter). Because it drives a
full browser, scrape intervals should be long and timeouts high.

In `prometheus.yml`:

```yaml
  - job_name: 'login_exporter'
    scrape_interval: 5m
    scrape_timeout: 90s
    metrics_path: /probe
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: 127.0.0.1:9980
    file_sd_configs:
      - refresh_interval: 2m
        files:
          - '/etc/prometheus/login_targets.json'
```

In `login_targets.json`:

```json
[
  {
    "labels": {
      "group": "apps",
      "host": "hostname",
      "ip": "ip_address",
      "job": "login_exporter"
    },
    "targets": [
      "target_name_defined_in_login_yml"
    ]
  }
]
```

## Development

This application is open-source and can be extended. This repository is a mirror of our
internally-hosted repository, so pull requests cannot be merged directly. Pull requests are
still welcome — acceptable patches will be applied manually to the internal repo. You are also
free to fork this repository and maintain your own changes.

### Build

```bash
go build -o ./login_exporter
```

## Docker

A `Dockerfile` is included. Chrome is bundled in the image so no additional setup is needed.

```bash
docker build -t login-exporter .
docker run --rm -v /path/to/login.yml:/etc/prometheus/login.yml -p 9980:9980 login-exporter
```

## License

See the [LICENSE](LICENSE) file.