# hcloud-k8s

A CLI tool written in Go for bootstrapping Kubernetes clusters on Hetzner Cloud.

## Features

- Automated provisioning of Kubernetes clusters on Hetzner Cloud
- Interactive configuration via CLI prompts
- Support for customizing:
  - Cluster name
  - Region selection
  - Node count
  - Server types
- Automated master node deployment
- Infrastructure tagging for better resource management

## Prerequisites

- Go 1.21 or higher
- A Hetzner Cloud account
- Hetzner Cloud API token with read/write permissions

## Installation

```bash
go install github.com/danielbooth-cloud/hcloud-k8s
```

Or build from source:

```bash
git clone https://github.com/danielbooth-cloud/hcloud-k8s.git
cd hcloud-k8s
go build 
```

## Usage

1. Run the tool:
```bash
hcloud-k8s bootstrap
```

2. Follow the interactive prompts to configure your cluster:
   - Enter your Hetzner Cloud API token
   - Choose a cluster name
   - Select your desired region
   - Choose server types
   - Specify the number of nodes

The tool will then:
1. Validate your configuration
2. Create the required infrastructure
3. Configure the master nodes
4. Output connection information

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [hcloud-go](https://github.com/hetznercloud/hcloud-go)
- Command line interface powered by [cobra](https://github.com/spf13/cobra)
- Interactive prompts using [survey](https://github.com/AlecAivazis/survey)