local compose = import '../docker-compose.libsonnet';
local values = import 'values.jsonnet';

compose(values) + {
  // Overwrite with custom values
  services+: {
    'alertmanager-bot'+: {
      // ports: [
      //   '80:8080/tcp',
      // ],
      volumes: [
        './data:/data',
      ],
    },
  },
}
