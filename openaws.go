package main

/* openaws.go
 *
 * Copyright (C) 2021 Mitsutaka Kato
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

type Profile struct {
	SourceProfile     string
	AssumingRoleName  string
	AssumingAccountID string
	ConsoleUserName   string
	ConsolePassword   string
	AccountID         string
}

const url = "https://console.aws.amazon.com/console/home"

func main() {

	profs := map[string]Profile{}

	homedir, _ := os.UserHomeDir()

	var sourceProfile Profile
	var assumingProfile Profile

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

		p := Profile{
			SourceProfile: sections[i].Key("source_profile").Value(),
		}

		if len(roleArn) == 6 {
			p.AssumingRoleName = strings.Replace(roleArn[5], "role/", "", 1)
			p.AssumingAccountID = roleArn[4]
		}

		// for this time, get console username and password from the .aws/config
		//TODO: organize well
		if sections[i].HasKey("console_account") {
			p.AccountID = sections[i].Key("console_account").Value()
			p.ConsoleUserName = sections[i].Key("console_username").Value()
			p.ConsolePassword = sections[i].Key("console_password").Value()
		}

		profs[pname] = p
		fmt.Printf("%s : %s\n", pname, sections[i].Key("role_arn"))
	}

	fmt.Print("Profile Name: ")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	prof := strings.Replace(text, "\n", "", -1)
	sourceProfile = profs[prof]
	assumingProfile = profs[prof]

	if assumingProfile.SourceProfile != "" {
		sourceProfile = profs[assumingProfile.SourceProfile]
	}

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
	page.Find("#resolving_input").SendKeys(sourceProfile.AccountID)
	page.Find("#next_button").Click()

	fmt.Fprintf(os.Stderr, "entering credentials\n")
	page.Find("#username").SendKeys(sourceProfile.ConsoleUserName)
	page.Find("#password").SendKeys(sourceProfile.ConsolePassword)
	page.Find("#signin_button").Click()

	fmt.Fprintf(os.Stderr, "waiting for login...\n")

	if assumingProfile.SourceProfile == "" {
		return
	}

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
	page.FindByID("account").Fill(profs[prof].AssumingAccountID)
	page.FindByID("roleName").Fill(profs[prof].AssumingRoleName)
	page.FindByID("input_switchrole_button").Click()
}
