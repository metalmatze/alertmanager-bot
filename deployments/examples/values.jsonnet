{
  name: 'alertmanager-bot',
  image: 'metalmatze/alertmanager-bot:0.4.2',

  alertmanager: {
    url: 'http://localhost:9093',
  },

  // This is just an example!
  // Remove this from jsonnet and create a proper secret!
  telegram: {
    admin: '1234',
    token: 'XXXXXXX',
  },

  ports: {
    http: 8080,
  },

  listen: {
    addr: '0.0.0.0:%d' % $.ports.http,
  },

  log: {
    level: 'info',  // debug info warn error
    json: false,
  },

  storage: {
    bolt: {
      path: '/data/bot.db',
    },
    // consul: {
    //   url: 'localhost:8500',
    // },
  },

  template: {
    path: '/templates/default.tmpl',
  },

  // Kubernetes only

  metadata+: {
    namespace: 'monitoring',
  },
  replicas: 1,
  resources: {
    limits: {
      cpu: '100m',
      memory: '128Mi',
    },
    requests: {
      cpu: '25m',
      memory: '64Mi',
    },
  },
  pvc: {
    size: '1Gi',
    class: 'standard',
  },
}
