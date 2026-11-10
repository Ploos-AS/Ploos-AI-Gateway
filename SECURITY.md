# Security policy

The gateway is in early development and is not production-ready.

Please do not open public issues with credentials, customer data or exploitable vulnerability details. Contact the maintainers privately through GitHub's private vulnerability reporting feature, if enabled, or through a secure private channel listed by the repository owner.

M0 implements no AI model routing and no authentication. It binds to loopback by default outside containers. Never expose M0 publicly.

Providers' data handling, retention and regional processing depend on the upstream service. Do not assume that hosting this gateway in Europe ensures EU-only model processing.
