package main

/* openaws.go
 *
 * Copyright (C) 2021 Mitsutaka Naito
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or (at
 * your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301, USA.
 */

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"os"

	"github.com/sclevine/agouti"
	"gopkg.in/ini.v1"
)

type Role struct {
	RoleName  string
	AccountID string
}

const url = "https://console.aws.amazon.com/console/home"

func main() {

	profs := map[string]Role{}

	homedir, _ := os.UserHomeDir()

	var aws string
	var username string
	var password string

	// Check Chrome installation
	{
		if runtime.GOOS == "darwin" {
			{
				cmd := exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--version")
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					log.Fatal(err)
				}
				if err := cmd.Start(); err != nil {
					log.Fatal(err)
				}
				version, _ := bufio.NewReader(stdout).ReadString('\n')
				fmt.Fprintf(os.Stderr, "Your Google Chrome version: %s\n", version)
			}

			cmd := exec.Command("chromedriver", "--version")
			_, err := cmd.StdoutPipe()
			if err != nil {
				log.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "`chromedriver` not found.\n   Visit https://chromedriver.chromium.org/ to get driver.")
				log.Fatal(err)
			}
		}
	}

	// Obtain AWS Profiles
	opt := ini.LoadOptions{
		UnescapeValueDoubleQuotes: true,
	}

	cfg, _ := ini.LoadSources(opt, filepath.Join(homedir, ".aws", "config"))
	sections := cfg.Sections()
	for i := range sections {
		pname := strings.Replace(sections[i].Name(), "profile ", "", 1)
		roleArn := strings.Split(sections[i].Key("role_arn").Value(), ":")

		// for this time, get console username and password from the .aws/config
		//TODO: organize well
		if sections[i].HasKey("console_account") {
			aws = sections[i].Key("console_account").Value()
			username = sections[i].Key("console_username").Value()
			password = sections[i].Key("console_password").Value()
		}

		if len(roleArn) < 5 {
			continue
		}

		p := Role{
			RoleName:  strings.Replace(roleArn[5], "role/", "", 1),
			AccountID: roleArn[4],
		}

		profs[pname] = p
		fmt.Printf("%s : %s\n", pname, sections[i].Key("role_arn"))
	}

	fmt.Print("Profile Name: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	prof := strings.Replace(text, "\n", "", -1)

	// launch Selenium driver
	options := agouti.ChromeOptions(
		"args", []string{
			"--disable-gpu",
		})

	// we don't want to close driver after login.
	driver := agouti.ChromeDriver(options)
	driver.Start()

	page, err := driver.NewPage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "opening AWS...\n")
	page.Navigate(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	// enter login credentials...
	fmt.Fprintf(os.Stderr, "entering account ID\n")
	page.Find("#iam_user_radio_button").Click()
	page.Find("#resolving_input").SendKeys(aws)
	page.Find("#next_button").Click()

	fmt.Fprintf(os.Stderr, "entering credentials\n")
	page.Find("#username").SendKeys(username)
	page.Find("#password").SendKeys(password)
	page.Find("#signin_button").Click()

	fmt.Fprintf(os.Stderr, "waiting for MFA...\n")

	// wait for browser title change
	for {
		title, _ := page.Title()
		time.Sleep(100 * time.Millisecond)
		if title == "AWS Management Console" {
			time.Sleep(1 * time.Second)
			break
		}
	}

	// visit Switch Role
	fmt.Fprintf(os.Stderr, "ok. assuming a role\n")
	page.FindByID("nav-usernameMenu").Click()
	page.FindByLink("Switch Roles").Click()
	page.FindByID("switchrole_firstrun_button").Click()

	// do the switch role
	page.FindByID("account").Fill(profs[prof].AccountID)
	page.FindByID("roleName").Fill(profs[prof].RoleName)
	page.FindByID("input_switchrole_button").Click()
}
