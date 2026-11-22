# Security policy

This application shares one clipboard with all clients that can reach it.
The default Docker configuration publishes the service on the local host.
For access outside a trusted network, use a VPN or an authenticated reverse
proxy in front of the entire application, including its WebSocket endpoint.
See the [deployment guide](docs/deployment.md) for configuration.

Security fixes are made on the current release line. Report potential
vulnerabilities privately using the repository's **Security → Report a
vulnerability** form. Include the affected version, deployment settings and
steps to reproduce with sample data. Keep credentials and actual clipboard
content out of the report.
