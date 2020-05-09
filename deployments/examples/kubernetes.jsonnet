local kubernetes = import '../kubernetes.libsonnet';
local values = import 'values.jsonnet';

local k = kubernetes(values) + {
  // Overwrite or add thing here
  statefulSet+: {
    spec+: {
      template+: {
        spec+: {
          restartPolicy: 'Always',
        },
      },
    },
  },
};

{
  apiVersion: 'v1',
  kind: 'List',
  items: [
    k[object]
    for object in std.objectFields(k)
  ],
}
