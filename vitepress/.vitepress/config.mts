import { defineConfig } from 'vitepress'

export default defineConfig({
  title: "MageBox",
  description: "Fast, native Magento development environment",

  ignoreDeadLinks: [
    /^http:\/\/localhost/,
    /^http:\/\/127\.0\.0\.1/
  ],

  head: [
    ['link', { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32x32.png' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16x16.png' }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#f26322' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'MageBox' }],
    ['meta', { property: 'og:description', content: 'Fast, native Magento development environment' }],
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Guide', link: '/guide/what-is-magebox' },
      { text: 'Services', link: '/services/nginx' },
      { text: 'Testing', link: '/guide/testing-tools' },
      { text: 'Teams', link: '/guide/teams' },
      { text: 'Reference', link: '/reference/commands' },
      { text: 'About', link: '/about' },
      {
        text: 'v0.14.1',
        items: [
          { text: 'Changelog', link: '/changelog' },
          { text: 'Roadmap', link: '/roadmap' },
          { text: 'GitHub', link: 'https://github.com/qoliber/magebox' }
        ]
      }
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is MageBox?', link: '/guide/what-is-magebox' },
            { text: 'Why MageBox?', link: '/guide/why-magebox' },
            { text: 'Architecture', link: '/guide/architecture' }
          ]
        },
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Bootstrap', link: '/guide/bootstrap' }
          ]
        },
        {
          text: 'Migration Guides',
          items: [
            { text: 'From Warden', link: '/guide/migrating-from-warden' },
            { text: 'From DDEV', link: '/guide/migrating-from-ddev' },
            { text: 'From Valet/Valet+', link: '/guide/migrating-from-valet' },
            { text: 'From Herd', link: '/guide/migrating-from-herd' }
          ]
        },
        {
          text: 'Configuration',
          items: [
            { text: 'Project Config (.magebox.yaml)', link: '/guide/project-config' },
            { text: 'Global Config', link: '/guide/global-config' },
            { text: 'Local Overrides', link: '/guide/local-overrides' },
            { text: 'PHP INI Settings', link: '/guide/php-ini' }
          ]
        },
        {
          text: 'Services',
          items: [
            { text: 'Nginx', link: '/services/nginx' },
            { text: 'PHP-FPM', link: '/services/php-fpm' },
            { text: 'Database (MySQL/MariaDB)', link: '/services/database' },
            { text: 'Redis', link: '/services/redis' },
            { text: 'OpenSearch/Elasticsearch', link: '/services/opensearch' },
            { text: 'RabbitMQ', link: '/services/rabbitmq' },
            { text: 'Mailpit', link: '/services/mailpit' },
            { text: 'Varnish', link: '/services/varnish' },
            { text: 'Blackfire', link: '/services/blackfire' },
            { text: 'Tideways', link: '/services/tideways' }
          ]
        },
        {
          text: 'Team Collaboration',
          items: [
            { text: 'Overview', link: '/guide/teams' }
          ]
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Multi-Domain Setup', link: '/guide/multi-domain' },
            { text: 'SSL Certificates', link: '/guide/ssl' },
            { text: 'DNS Configuration', link: '/guide/dns' },
            { text: 'Custom Commands', link: '/guide/custom-commands' },
            { text: 'Multiple Projects', link: '/guide/multiple-projects' },
            { text: 'CLI Wrappers', link: '/guide/php-wrapper' },
            { text: 'Xdebug', link: '/guide/xdebug' },
            { text: 'Testing & Code Quality', link: '/guide/testing-tools' },
            { text: 'Logs & Reports', link: '/guide/logs' },
            { text: 'Admin Commands', link: '/guide/admin' },
            { text: 'Linux Installers', link: '/guide/linux-installers' },
            { text: 'Integration Testing', link: '/guide/testing' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/commands' },
            { text: 'Configuration Options', link: '/reference/config-options' },
            { text: 'Service Ports', link: '/reference/ports' }
          ]
        },
        {
          text: 'Help',
          items: [
            { text: 'Troubleshooting', link: '/guide/troubleshooting' }
          ]
        }
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/commands' },
            { text: 'Configuration Options', link: '/reference/config-options' },
            { text: 'Service Ports', link: '/reference/ports' },
            { text: 'Environment Variables', link: '/reference/environment' }
          ]
        }
      ],
      '/services/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is MageBox?', link: '/guide/what-is-magebox' },
            { text: 'Why MageBox?', link: '/guide/why-magebox' },
            { text: 'Architecture', link: '/guide/architecture' }
          ]
        },
        {
          text: 'Getting Started',
          items: [
            { text: 'Installation', link: '/guide/installation' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Bootstrap', link: '/guide/bootstrap' }
          ]
        },
        {
          text: 'Migration Guides',
          items: [
            { text: 'From Warden', link: '/guide/migrating-from-warden' },
            { text: 'From DDEV', link: '/guide/migrating-from-ddev' },
            { text: 'From Valet/Valet+', link: '/guide/migrating-from-valet' },
            { text: 'From Herd', link: '/guide/migrating-from-herd' }
          ]
        },
        {
          text: 'Configuration',
          items: [
            { text: 'Project Config (.magebox.yaml)', link: '/guide/project-config' },
            { text: 'Global Config', link: '/guide/global-config' },
            { text: 'Local Overrides', link: '/guide/local-overrides' },
            { text: 'PHP INI Settings', link: '/guide/php-ini' }
          ]
        },
        {
          text: 'Services',
          items: [
            { text: 'Nginx', link: '/services/nginx' },
            { text: 'PHP-FPM', link: '/services/php-fpm' },
            { text: 'Database (MySQL/MariaDB)', link: '/services/database' },
            { text: 'Redis', link: '/services/redis' },
            { text: 'OpenSearch/Elasticsearch', link: '/services/opensearch' },
            { text: 'RabbitMQ', link: '/services/rabbitmq' },
            { text: 'Mailpit', link: '/services/mailpit' },
            { text: 'Varnish', link: '/services/varnish' },
            { text: 'Blackfire', link: '/services/blackfire' },
            { text: 'Tideways', link: '/services/tideways' }
          ]
        },
        {
          text: 'Team Collaboration',
          items: [
            { text: 'Overview', link: '/guide/teams' }
          ]
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Multi-Domain Setup', link: '/guide/multi-domain' },
            { text: 'SSL Certificates', link: '/guide/ssl' },
            { text: 'DNS Configuration', link: '/guide/dns' },
            { text: 'Custom Commands', link: '/guide/custom-commands' },
            { text: 'Multiple Projects', link: '/guide/multiple-projects' },
            { text: 'CLI Wrappers', link: '/guide/php-wrapper' },
            { text: 'Xdebug', link: '/guide/xdebug' },
            { text: 'Testing & Code Quality', link: '/guide/testing-tools' },
            { text: 'Logs & Reports', link: '/guide/logs' },
            { text: 'Admin Commands', link: '/guide/admin' },
            { text: 'Linux Installers', link: '/guide/linux-installers' },
            { text: 'Integration Testing', link: '/guide/testing' }
          ]
        },
        {
          text: 'Reference',
          items: [
            { text: 'CLI Commands', link: '/reference/commands' },
            { text: 'Configuration Options', link: '/reference/config-options' },
            { text: 'Service Ports', link: '/reference/ports' }
          ]
        },
        {
          text: 'Help',
          items: [
            { text: 'Troubleshooting', link: '/guide/troubleshooting' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/qoliber/magebox' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 qoliber'
    },

    search: {
      provider: 'local'
    },

    editLink: {
      pattern: 'https://github.com/qoliber/magebox/edit/main/vitepress/:path',
      text: 'Edit this page on GitHub'
    }
  }
})
