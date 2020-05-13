function(params) {
  local bot = {
    // Defining all the defaults
    local b = self,
    name: 'alertmanager-bot',
    image: 'metalmatze/alertmanager-bot:0.4.2',
    alertmanager: {
      url: 'http://localhost:9093',
    },
    ports: {
      http: 8080,
    },
    log: {
      level: 'info',
      json: false,
    },
    storage: error 'please provide the storage configuration',

    // Set defaults for Kubernetes from the defaults above
    metadata: {
      name: b.name,
      namespace: 'monitoring',
      labels: {
        'app.kubernetes.io/name': b.name,
      },
    },
    replicas: 1,
    resources: {},
  } + params,

  service: {
    apiVersion: 'v1',
    kind: 'Service',
    metadata: bot.metadata,
    spec: {
      ports: [
        { name: name, port: bot.ports[name], targetPort: bot.ports[name] }
        for name in std.objectFields(bot.ports)
      ],
      selector: bot.metadata.labels,
    },
  },

  secret: if std.objectHas(bot, 'telegram') then {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: bot.metadata,
    type: 'Opaque',
    data: {
      admin: std.base64(bot.telegram.admin),
      token: std.base64(bot.telegram.token),
    },
  } else null,

  statefulSet: {
    apiVersion: 'apps/v1',
    kind: 'StatefulSet',
    metadata: bot.metadata,
    spec: {
      podManagementPolicy: 'OrderedReady',
      replicas: bot.replicas,
      selector: {
        matchLabels: bot.metadata.labels,
      },
      serviceName: bot.metadata.name,
      template: {
        metadata: bot.metadata,
        spec: {
          containers: [
            {
              name: bot.name,
              image: bot.image,
              imagePullPolicy: 'IfNotPresent',
              args: [
                '--alertmanager.url=%s' % bot.alertmanager.url,
                '--log.level=%s' % bot.log.level,
              ] + (
                if bot.log.json then ['--log.json'] else []
              ) + (
                if std.objectHas(bot.storage, 'bolt') then [
                  '--store=bolt',
                  '--bolt.path=%s' % bot.storage.bolt.path,
                ] else []
              ) + (
                if std.objectHas(bot.storage, 'consul') then [
                  '--store=consul',
                  '--consul.url=%s' % bot.storage.consul.url,
                ] else []
              ),
              env: if std.objectHas(bot, 'telegram') then [
                {
                  name: 'TELEGRAM_ADMIN',
                  valueFrom: { secretKeyRef: {
                    name: bot.metadata.name,
                    key: 'admin',
                  } },
                },
                {
                  name: 'TELEGRAM_TOKEN',
                  valueFrom: { secretKeyRef: {
                    name: bot.metadata.name,
                    key: 'token',
                  } },
                },
              ] else [],
              ports: [
                { name: name, containerPort: bot.ports[name] }
                for name in std.objectFields(bot.ports)
              ],
              resources: bot.resources,
              volumeMounts: [
                { mountPath: '/data', name: 'data' },
              ],
            },
          ],
          volumes: if std.objectHas(bot, 'pvc') then [
            { name: 'data', persistentVolumeClaim: { claimName: 'data' } },
          ] else [
            { name: 'data', emptyDir: {} },
          ],
        },
      },
      volumeClaimTemplates: if std.objectHas(bot, 'pvc') then [
        {
          apiVersion: 'v1',
          kind: 'PersistentVolumeClaim',
          metadata: bot.metadata {
            name: 'data',
          },
          spec: {
            accessModes: ['ReadWriteOnce'],
            resources: { requests: { storage: bot.pvc.size } },
            storageClassName: bot.pvc.class,
          },
        },
      ] else [],
    },
  },
}
