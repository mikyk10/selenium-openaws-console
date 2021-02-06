# selenium-openaws-console
Automation tool for AWS console login.


## Overview
Assuming you are working on multiple AWS consoles simultaneously across different account IDs, you might switch IAM role(s) back and forth. Eventually, the role history would be fully consumed. The first IAM role you have used is out of history. You may even attempt to open a number of browser windows and login, but it fails because these windows share the same session.

This tool can open up AWS console windows across multiple AWS accounts without boring and repetitive ID/PW input.

# Prerequisites

macOS
Go 1.15
Google Chrome
Selenium webdriver for Chrome

## Install


## Configuring

The first thing you need to do is to add your credentials to ~/.aws/config with special syntax for this app.
username and password should be double-quoted so you can use special characters in the password.
The special configuration syntax have not any special meaning in the AWS CLI tools.

```
[profile me]
console_username = "ENTER_YOUR_USERNAME"
console_password = "ENTER_YOUR_PASSWORD"
```

## Usage

Just enter the command. You are asked to choose an AWS profile name then it 


## Contributing

Contributions welcome! Please read the [contributing guidelines](CONTRIBUTING.md) first.


## License

[GPLv3](LICENSE)
