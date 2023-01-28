package setup

import (
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"text/template"

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

//go:embed templates/launchdaemon.tpl
var launchDaemonTemplate string

type spec struct {
	Name    string
	Project string
	Network struct {
		Domain     string
		Subdomains []struct {
			Address string
			Names   []string
		}
	}
}

func RunSetup() {
	isRoot := isRoot()
	uid := os.Getgid()
	gid := os.Getuid()

	if !isRoot {
		log.Fatal(color.RedString("The setup command must be ran with sudo. Please rerun with sudo"))
	}

	// Run the initial checks
	fmt.Println(color.CyanString("\nRunning system checks..."))
	checks()
	fmt.Println(color.GreenString("✅ Checks complete."))

	// Load the spec
	var s spec
	s.load()

	// Setup the networking
	fmt.Println(color.CyanString("\nCreating Launch Daemons..."))
	launchDaemons(s)
	fmt.Println(color.GreenString("✅ Launch Daemons created and started."))

	fmt.Println(color.CyanString("\nCreating host file entries..."))
	hostsFile(s)
	fmt.Println(color.GreenString("✅ Host file entries created"))

	fmt.Println(color.CyanString("\nSetting up the docker networks..."))
	dockerNetworks(s, uid, gid)
	fmt.Println(color.GreenString("✅ Docker networks created"))

}

func checks() {
	// Docker
	fmt.Print("Checking for Docker... ")
	_, err := exec.LookPath("docker")
	if err != nil {
		log.Fatal(color.RedString(
			`Oh no! Docker is not installed on your machine. 
			Please download Docker Desktop before continuing at: 
			https://www.docker.com/products/docker-desktop/`))
	}
	fmt.Println(color.CyanString("🐋 Docker installation detected."))

	// mkcert
	fmt.Print("Checking for mkcert... ")
	_, err = exec.LookPath("mkcert")
	if err != nil {
		log.Fatal(`mkcert is not installed. Please install mkcert with 'brew install mkcert'.`)
	}
	fmt.Println(color.CyanString("🔏 mkcert installation detected."))
}

func (c *spec) load() *spec {
	spec := "./mlspec.yaml"

	if _, err := os.Stat(spec); err == nil {
		yamlFile, err := os.ReadFile(spec)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}
		err = yaml.Unmarshal(yamlFile, c)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
		return c

	} else {
		log.Fatal("mlspec.yaml file not found in current directory. Aborting.")
		return nil
	}
}

func dockerNetworks(s spec, uid int, gid int) {
	networks := []string{fmt.Sprintf("%s-net", strings.ToLower(s.Project)), "monitoring"}
	existing, err := exec.Command("docker", "network", "list").Output()
	if err != nil {
		log.Fatal(err)
	}

	for _, network := range networks {
		// Check if network exists
		if !strings.Contains(string(existing), network) {
			fmt.Println("Creating docker network", network)
			cmd := exec.Command("docker", "network", "create", network)
			cmd.SysProcAttr = &syscall.SysProcAttr{}
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

			if errors.Is(cmd.Err, exec.ErrDot) {
				cmd.Err = nil
			}
			if err := cmd.Run(); err != nil {
				log.Fatal(err)
			}
		} else {
			fmt.Println("Network", network, "already exists. Skipping...")
		}
	}
}

/**
* Setup the Launch Daemon based on the IP Address and name
 */
func launchDaemons(s spec) {
	ldPath := "/Library/LaunchDaemons"

	// For each subdomain in the template, create the launch daemon
	for index, sd := range s.Network.Subdomains {
		daemonName := fmt.Sprintf("com.mosaiclearning.%s-%d-dev.plist", s.Name, index)
		daemonPath := fmt.Sprintf("%s/%s", ldPath, daemonName)

		// First, check for existing file that matches what we want
		// skip := false
		// if existingFile, err := os.ReadFile(daemon); err == nil {
		// 	// Is it one we've created?
		// 	skip = containsIp(daemon, existingFile)
		// 	log.Fatal(color.RedString("The file " + daemon + " already exists."))
		// }

		files, err := os.ReadDir(ldPath)
		if err != nil {
			log.Fatal(err)
		}

		for _, file := range files {
			buff, err := os.ReadFile(ldPath + "/" + file.Name())
			if err != nil {
				panic(err)
			}
			s := string(buff)

			// Some Launch Daemon has taken up this IP Address
			if strings.Contains(s, sd.Address) {
				if file.Name() == daemonName {
					fmt.Println(color.HiCyanString("Correct Launch Daemon already exists. Skipping..."))
				} else {
					log.Fatal(color.RedString("IP Address " + sd.Address + " is already taken by " + file.Name()))
				}
			}
		}

		// Create a new launch template
		tmpl, err := template.New("todos").Parse(launchDaemonTemplate)
		if err != nil {
			log.Fatal(err)
		}

		file, err := os.Create(daemonPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		err = os.Chmod(daemonPath, 0755)
		if err != nil {
			log.Fatal(err)
		}

		// Apply the template to the vars map and write the result to file.
		vars := make(map[string]interface{})
		vars["IpAddress"] = sd.Address
		vars["Name"] = s.Name
		vars["Index"] = index
		tmpl.Execute(file, vars)

		// Load the launchdaemon
		exec.Command("launchctl", "load", daemonPath)
	}
}

type entry struct {
	IP    string
	Entry string
	Skip  bool
}

func hostsFile(s spec) {
	var entries []entry

	// Build out the hosts entries we need
	for _, sds := range s.Network.Subdomains {
		var e entry
		entry := sds.Address
		for _, sub := range sds.Names {
			entry += fmt.Sprintf(" %s.%s", sub, s.Network.Domain)
		}
		e.IP = sds.Address
		e.Entry = entry
		e.Skip = false
		entries = append(entries, e)
	}

	// Read in the hosts file for scanning
	hostsfile := "/etc/hosts"
	b, err := ioutil.ReadFile(hostsfile)
	if err != nil {
		panic(err)
	}
	hosts := strings.Split(string(b), "\n")

	for hostI, host := range hosts {
		for entryI, entry := range entries {
			if strings.Contains(host, entry.IP) {
				// Check if match for existing
				if host == entry.Entry {
					fmt.Println("Correct entry exists, skipping...")
					entries[entryI].Skip = true
				} else if strings.Contains(host, s.Network.Domain) {
					fmt.Println("Updating existing entry...")
					hosts[hostI] = entry.Entry
					entries[entryI].Skip = true
				} else {
					// Someone else took this entry. Fatal
					log.Fatal(color.RedString("IP Address " + entry.IP + "is already in use by another project. Please remove the conflicting entry from your hosts file to continue."))
				}
			}
		}
	}

	for _, entry := range entries {
		// Add any entries that weren't accounted for
		if !entry.Skip {
			fmt.Println("Adding missing entry...")
			hosts = append(hosts, entry.Entry)
		}
	}

	// Write the file back out
	output := strings.Join(hosts, "\n")
	err = ioutil.WriteFile(hostsfile, []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}

}

func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("[isRoot] Unable to get current user: %s", err)
	}
	return currentUser.Username == "root"
}
