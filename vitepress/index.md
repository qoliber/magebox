---
layout: home

hero:
  name: "MageBox"
  text: "Native Magento Development"
  tagline: Native PHP and Nginx with Docker services.
  image:
    src: /logo.svg
    alt: MageBox
  actions:
    - theme: brand
      text: Get Started
      link: /guide/what-is-magebox
    - theme: alt
      text: View on GitHub
      link: https://github.com/qoliber/magebox

features:
  - icon: 🚀
    title: Native PHP/Nginx
    details: PHP-FPM and Nginx run on your machine. Direct file access.

  - icon: 🐳
    title: Docker Services
    details: MySQL, Redis, OpenSearch, RabbitMQ in containers.

  - icon: 🔄
    title: Multi-PHP
    details: Switch PHP versions per project. Run 8.1 and 8.4 simultaneously.

  - icon: 🔒
    title: Auto SSL
    details: HTTPS with mkcert for all .test domains.

  - icon: 📦
    title: Project Discovery
    details: See all projects with `magebox list`.

  - icon: 👥
    title: Team Collaboration
    details: Share configs, repos, and assets with `magebox fetch`.
---

## Quick Start

```bash
# Install
brew install qoliber/magebox/magebox

# First-time setup
magebox bootstrap

# Create a new project
magebox new mystore
```

## Supported Services

- **Database**: MySQL 5.7, 8.0, 8.4 / MariaDB 10.4, 10.6, 11.4
- **Cache**: Redis
- **Search**: OpenSearch 2.x / Elasticsearch 7.x, 8.x
- **Queue**: RabbitMQ
- **Mail**: Mailpit
- **HTTP Cache**: Varnish

## Platform Support

- macOS (Intel & Apple Silicon)
- Linux (Ubuntu, Debian, Fedora, Arch)
- Windows WSL2
