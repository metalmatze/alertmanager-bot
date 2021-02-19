function(params) {
  local bot = {
    log: {
      level: 'info',
      json: false,
    },
  } + params,

  version: '3',
  networks: {
    'alertmanager-bot': {},
  },
  services: {
    'alertmanager-bot': {
      image: bot.image,
      restart: 'always',
      networks: ['alertmanager-bot'],
      environment: if std.objectHas(bot, 'telegram') then {
        TELEGRAM_ADMIN: bot.telegram.admin,
        TELEGRAM_TOKEN: bot.telegram.token,
      } else {},
      command: [
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
      ports: [
        '%d:%d' % [bot.ports[name], bot.ports[name]]
        for name in std.objectFields(bot.ports)
      ],
    },
  },
}
