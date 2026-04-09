import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'mememory',
  description: 'Persistent memory for AI agents',
  base: '/mememory/',
  ignoreDeadLinks: [/localhost/],
  appearance: false,
  head: [
    ['link', { rel: 'preconnect', href: 'https://fonts.googleapis.com' }],
    ['link', { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' }],
    ['link', { href: 'https://fonts.googleapis.com/css2?family=Wix+Madefor+Display:wght@400;500;600;700;800&display=swap', rel: 'stylesheet' }],
  ],
  themeConfig: {
    logo: undefined,
    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'Reference', link: '/reference/mcp-tools' },
      { text: 'GitHub', link: 'https://github.com/scott-walker/mememory' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is mememory?', link: '/guide/what-is-mememory' },
            { text: 'Getting Started', link: '/guide/getting-started' },
          ],
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Memory Model', link: '/guide/memory-model' },
            { text: 'Scopes & Hierarchy', link: '/guide/scopes' },
            { text: 'Scoring & Recall', link: '/guide/scoring' },
            { text: 'Session Bootstrap', link: '/guide/bootstrap' },
          ],
        },
        {
          text: 'Configuration',
          items: [
            { text: 'Environment Variables', link: '/guide/configuration' },
            { text: 'Embedding Providers', link: '/guide/embedding-providers' },
            { text: 'MCP Client Setup', link: '/guide/mcp-client-setup' },
          ],
        },
        {
          text: 'Advanced',
          items: [
            { text: 'Architecture', link: '/guide/architecture' },
            { text: 'Backup & Migration', link: '/guide/backup' },
            { text: 'Contributing', link: '/guide/contributing' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'MCP Tools', link: '/reference/mcp-tools' },
            { text: 'CLI Commands', link: '/reference/cli' },
            { text: '.mememory File', link: '/reference/mememory-file' },
            { text: 'Admin API', link: '/reference/admin-api' },
            { text: 'Changelog', link: '/reference/changelog' },
          ],
        },
      ],
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/scott-walker/mememory' },
    ],
    search: {
      provider: 'local',
    },
    footer: {
      message: 'Released under the MIT License.',
    },
  },
})
