---
layout: home

hero:
  name: "MageBox"
  text: "Native Magento Development"
  tagline: For individuals and teams. Full native speed. Zero container overhead.
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
    title: Native Performance
    details: PHP-FPM and Nginx run natively on your machine. No file sync overhead, no container layers. Your code changes are instant.

  - icon: 🐳
    title: Smart Docker Usage
    details: Docker only for stateless services - MySQL, Redis, OpenSearch, RabbitMQ. The best of both worlds without the downsides.

  - icon: 🔄
    title: Multi-PHP Support
    details: Switch PHP versions per project instantly. Run PHP 8.1 on one project, 8.4 on another. No rebuilding containers.

  - icon: 🔒
    title: Auto SSL
    details: HTTPS works out of the box with mkcert. All your .test domains get valid local SSL certificates automatically.

  - icon: 📦
    title: Project Discovery
    details: MageBox automatically discovers all your projects. Run `magebox list` to see everything at a glance.

  - icon: ⚡
    title: One Command Setup
    details: Run `magebox bootstrap` once, then `magebox start` in any project. That's it. No complex configuration needed.

  - icon: 👥
    title: Team Collaboration
    details: Share project configs, Git repos (GitHub/GitLab/Bitbucket), and asset storage. New team member? Run `magebox fetch` and start coding.
---

## Quick Start

```bash
# Install MageBox (choose one)
brew install qoliber/magebox/magebox    # Homebrew
curl -fsSL https://get.magebox.dev | bash # or install script

# First-time setup
magebox bootstrap

# Initialize a project
cd /path/to/magento
magebox init mystore

# Start development
magebox start
```

## Why MageBox?

| Feature | How It Works |
|---------|--------------|
| Native PHP/Nginx | Direct file access, no sync layer |
| Docker services | MySQL, Redis, OpenSearch in containers |
| Multi-PHP support | Instant switching between versions |
| Fast startup | Projects start in ~2 seconds |
| Low memory | Native PHP shares resources efficiently |

## Supported Services

- **Database**: MySQL 5.7, 8.0, 8.4 / MariaDB 10.4, 10.6, 11.4
- **Cache**: Redis
- **Search**: OpenSearch 2.x / Elasticsearch 7.x, 8.x
- **Queue**: RabbitMQ
- **Mail**: Mailpit
- **HTTP Cache**: Varnish

## Platform Support

- macOS (Intel & Apple Silicon)
- Linux (Ubuntu, Debian, Fedora)
- Windows WSL2 (Ubuntu, Fedora)
