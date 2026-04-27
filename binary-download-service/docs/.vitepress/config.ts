import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Node Push Exporter',
  description: 'Node 指标推送服务文档',
  srcDir: '.',

  ignoreDeadLinks: true,

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],

  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/' },
      { text: '指标文档', link: '/metrics/' }
    ],

    sidebar: {
      '/guide/': [
        {
          text: '指南',
          items: [
            { text: '概述', link: '/guide/' },
            { text: '安装部署', link: '/guide/install' },
            { text: '快速开始', link: '/guide/quickstart' },
            // { text: '用户手册', link: '/guide/usage' }
          ]
        }
      ],
      '/api/': [
        {
          text: 'API 文档',
          items: [
            { text: '概述', link: '/api/' },
            { text: '文件管理', link: '/api/files' },
            { text: '节点管理', link: '/api/agents' },
            { text: 'Python 查询', link: '/api/python' },
            { text: '错误响应', link: '/api/errors' }
          ]
        }
      ],
      '/metrics/': [
        {
          text: '指标文档',
          items: [
            { text: '概述', link: '/metrics/' },
            { text: 'Prometheus 查询示例', link: '/metrics/prometheus' },
            { text: 'Node Exporter 指标', link: '/metrics/node-exporter' },
            { text: '自定义指标 (GPU)', link: '/metrics/gpu' }
          ]
        }
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/hankerbiao/ServerMetricPush' }
    ],

    search: {
      provider: 'local'
    }
  },

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  }
})