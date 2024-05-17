package cmd

import (
	"fmt"
	"github.com/bcdevtools/node-setup-check/utils"
	"github.com/spf13/cobra"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	flagRpcDomain     = "rpc"
	flagRpcPort       = "rpc-port"
	flagRestDomain    = "rest"
	flagRestPort      = "rest-port"
	flagJsonRpcDomain = "jsonrpc"
	flagJsonRpcPort   = "jsonrpc-port"
)

func GetGenNginxCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "gen-nginx",
		Short: "Generate nginx configuration",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			workingDir, err := os.Getwd()
			if err != nil {
				exitWithErrorMsgf("ERR: failed to get working directory: %v\n", err)
				return
			}

			rpcDomain, _ := cmd.Flags().GetString(flagRpcDomain)
			restDomain, _ := cmd.Flags().GetString(flagRestDomain)
			jsonRpcDomain, _ := cmd.Flags().GetString(flagJsonRpcDomain)
			rpcPort, _ := cmd.Flags().GetUint16(flagRpcPort)
			restPort, _ := cmd.Flags().GetUint16(flagRestPort)
			jsonRpcPort, _ := cmd.Flags().GetUint16(flagJsonRpcPort)

			// normalize
			normalizeEndpoint := func(endpoint string) string {
				return strings.TrimSpace(strings.ToLower(endpoint))
			}

			rpcDomain = normalizeEndpoint(rpcDomain)
			restDomain = normalizeEndpoint(restDomain)
			jsonRpcDomain = normalizeEndpoint(jsonRpcDomain)

			// validate domains

			isGenRpcConf := rpcDomain != ""
			isGenRestConf := restDomain != ""
			isGenJsonRpcConf := jsonRpcDomain != ""

			if !isGenRpcConf && !isGenRestConf && !isGenJsonRpcConf {
				exitWithErrorMsgf(
					"ERR: require at least one domain to generate, specify by flags --%s, --%s and --%s\n",
					flagRpcDomain, flagRestDomain, flagJsonRpcDomain,
				)
				return
			}

			validateDomain := func(domain string) {
				if domain == "" {
					return
				}

				if strings.Contains(domain, "://") {
					exitWithErrorMsgf("ERR: require domain! Should not contain protocol: %s\n", domain)
					return
				}

				if regexp.MustCompile(`:\d+$`).MatchString(domain) {
					exitWithErrorMsgf("ERR: require domain! Should not contain port: %s\n", domain)
					return
				}

				if strings.Contains(domain, "/") {
					exitWithErrorMsgf("ERR: require domain! Should not contain path: %s\n", domain)
					return
				}

				if !regexp.MustCompile(`^[a-z\d_-]+(\.[a-z\d_-]+)+$`).MatchString(domain) {
					exitWithErrorMsgf("ERR: invalid domain: %s\n", domain)
					return
				}

				if len(strings.Split(domain, ".")) < 2 {
					exitWithErrorMsgf(`ERR: bad naming domain [%s], must have at least one sub-domain.
Suggest: rpc/rest/json_rpc.testnet.my_chain.example.com`, domain)
					return
				}
			}

			var toBeGeneratedDomains []string
			if isGenRpcConf {
				validateDomain(rpcDomain)
				toBeGeneratedDomains = append(toBeGeneratedDomains, rpcDomain)
			}
			if isGenRestConf {
				validateDomain(restDomain)
				toBeGeneratedDomains = append(toBeGeneratedDomains, restDomain)
			}
			if isGenJsonRpcConf {
				validateDomain(jsonRpcDomain)
				toBeGeneratedDomains = append(toBeGeneratedDomains, jsonRpcDomain)
			}

			uniqueTracker := make(map[string]bool)
			for _, domain := range toBeGeneratedDomains {
				if uniqueTracker[domain] {
					exitWithErrorMsgf("ERR: duplicate domain: %s\n", domain)
					return
				}
				uniqueTracker[domain] = true
			}

			// validate ports

			validatePort := func(port uint16) {
				if port == 0 {
					exitWithErrorMsgf("ERR: require port to generate configuration\n")
					return
				}
				if port <= 1023 {
					exitWithErrorMsgf("ERR: port must be greater than 1023: %d\n", port)
					return
				}
			}

			if isGenRpcConf {
				validatePort(rpcPort)
			}
			if isGenRestConf {
				validatePort(restPort)
			}
			if isGenJsonRpcConf {
				validatePort(jsonRpcPort)
			}

			// check exists

			const fileSharedConf = "shared.conf"
			fileRpcConf := fmt.Sprintf("%s.conf", rpcDomain)
			fileRestConf := fmt.Sprintf("%s.conf", restDomain)
			fileJsonRpcConf := fmt.Sprintf("%s.conf", jsonRpcDomain)

			checkConfFileExists := func(file string) {
				_, exists, _, err := utils.FileInfo(file)
				if err != nil {
					exitWithErrorMsgf("ERR: failed to check if %s exists: %v\n", file, err)
					return
				}
				if exists {
					exitWithErrorMsgf("ERR: %s already exist\n", file)
					return
				}
			}

			checkConfFileExists(fileSharedConf)
			if isGenRpcConf {
				checkConfFileExists(fileRpcConf)
			}
			if isGenRestConf {
				checkConfFileExists(fileRestConf)
			}
			if isGenJsonRpcConf {
				checkConfFileExists(fileJsonRpcConf)
			}

			// generate

			writeSharedConfFile(fileSharedConf)
			if isGenRpcConf {
				writeRpcConfFile(rpcDomain, rpcPort, fileRpcConf)
			}
			if isGenRestConf {
				writeRestApiConfFile(restDomain, restPort, fileRestConf)
			}
			if isGenJsonRpcConf {
				writeJsonRpcConfFile(jsonRpcDomain, jsonRpcPort, fileJsonRpcConf)
			}

			fmt.Println("Generated nginx configuration files successfully:")
			fmt.Println("-", fileSharedConf)
			if isGenRpcConf {
				fmt.Println("-", fileRpcConf)
			}
			if isGenRestConf {
				fmt.Println("-", fileRestConf)
			}
			if isGenJsonRpcConf {
				fmt.Println("-", fileJsonRpcConf)
			}

			fmt.Println("\nGenerated! Copy these files to your nginx configuration directory and reload nginx")
			fmt.Printf("\n**WARN** Beware of overriding this file if you have existing configuration!!! sudo cp %s /etc/nginx/conf.d/\n", path.Join(workingDir, fileSharedConf))
			if isGenRpcConf {
				fmt.Printf("sudo cp %s /etc/nginx/conf.d/\n", path.Join(workingDir, fileRpcConf))
			}
			if isGenRestConf {
				fmt.Printf("sudo cp %s /etc/nginx/conf.d/\n", path.Join(workingDir, fileRestConf))
			}
			if isGenJsonRpcConf {
				fmt.Printf("sudo cp %s /etc/nginx/conf.d/\n", path.Join(workingDir, fileJsonRpcConf))
			}
			fmt.Println("sudo chown root:root /etc/nginx/conf.d/*.conf")
			fmt.Println("sudo chmod 644 /etc/nginx/conf.d/*.conf")
			fmt.Println("sudo nginx -t")
			fmt.Println("Finally reload nginx")
		},
	}

	cmd.Flags().String(flagRpcDomain, "", "Domain to expose Tendermint RPC")
	cmd.Flags().Uint16(flagRpcPort, 26657, "Port of Tendermint RPC to proxy")
	cmd.Flags().String(flagRestDomain, "", "Domain to expose Rest API")
	cmd.Flags().Uint16(flagRestPort, 1317, "Port of Rest API to proxy")
	cmd.Flags().String(flagJsonRpcDomain, "", "Domain to expose Ethereum Json-RPC")
	cmd.Flags().Uint16(flagJsonRpcPort, 8545, "Port of Ethereum Json-RPC to proxy")

	return cmd
}

func writeSharedConfFile(fileName string) {
	err := os.WriteFile(fileName, []byte(`
geo $limit {
    default 1;
}

map $limit $limit_key {
    0 "";
    1 $binary_remote_addr;
}

limit_req_zone $limit_key zone=req_zone:10m rate=60r/m;
`), 0644)

	if err != nil {
		exitWithErrorMsgf("ERR: failed to write shared conf file: %v\n", err)
		return
	}
}

func writeRpcConfFile(domain string, port uint16, fileName string) {
	upstreamName := fmt.Sprintf("upsr_%s", strings.ReplaceAll(domain, ".", "_"))

	//goland:noinspection HttpUrlsUsage
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(`
upstream %s {
    least_conn;
    server localhost:%d;
}

server {
    server_name %s;

    location / {
        limit_req zone=req_zone burst=10 nodelay;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain charset=UTF-8';
            add_header 'Content-Length' 0;
            return 204;
        }
        if ($request_method = 'POST') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
        }
        if ($request_method = 'GET') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
        }

        proxy_hide_header 'Access-Control-Allow-Origin';
        proxy_pass         http://%s;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection keep-alive;
        proxy_set_header   Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   X-Forwarded-Host $server_name;
    }

    location /websocket {
        proxy_pass http://%s/websocket;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }

    listen 80;
}
`, upstreamName, port, domain, upstreamName, upstreamName)), 0644)

	if err != nil {
		exitWithErrorMsgf("ERR: failed to write RPC conf file: %v\n", err)
		return
	}
}

func writeRestApiConfFile(domain string, port uint16, fileName string) {
	upstreamName := fmt.Sprintf("upsa_%s", strings.ReplaceAll(domain, ".", "_"))

	//goland:noinspection HttpUrlsUsage
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(`
upstream %s {
    least_conn;
    server localhost:%d;
}

server {
    server_name %s;

    location / {
        limit_req zone=req_zone burst=20 nodelay;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain charset=UTF-8';
            add_header 'Content-Length' 0;
            return 204;
        }
        if ($request_method = 'POST') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
        }
        if ($request_method = 'GET') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
        }

        proxy_hide_header 'Access-Control-Allow-Origin';
        proxy_pass         http://%s;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection keep-alive;
        proxy_set_header   Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   X-Forwarded-Host $server_name;
    }

    listen 80;
}`, upstreamName, port, domain, upstreamName)), 0644)

	if err != nil {
		exitWithErrorMsgf("ERR: failed to write RPC conf file: %v\n", err)
		return
	}
}

func writeJsonRpcConfFile(domain string, port uint16, fileName string) {
	upstreamName := fmt.Sprintf("upsj_%s", strings.ReplaceAll(domain, ".", "_"))

	//goland:noinspection HttpUrlsUsage
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(`
upstream %s {
    least_conn;
    server localhost:%d;
}

server {
    server_name %s;

    location / {
        limit_req zone=req_zone burst=5 nodelay;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
            add_header 'Access-Control-Max-Age' 1728000;
            add_header 'Content-Type' 'text/plain charset=UTF-8';
            add_header 'Content-Length' 0;
            return 204;
        }
        if ($request_method = 'POST') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS';
            add_header 'Access-Control-Allow-Headers' 'DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Origin,Accept,X-Server-Time';
        }

        proxy_hide_header 'Access-Control-Allow-Origin';
        proxy_pass         http://%s;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection keep-alive;
        proxy_set_header   Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_set_header   X-Forwarded-Host $server_name;
    }

    listen 80;
}
`, upstreamName, port, domain, upstreamName)), 0644)

	if err != nil {
		exitWithErrorMsgf("ERR: failed to write RPC conf file: %v\n", err)
		return
	}
}

func init() {
	rootCmd.AddCommand(GetGenNginxCmd())
}
