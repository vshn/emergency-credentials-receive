local tokens = std.extVar('tokens');
local server = std.extVar('server');

local clusterName = 'em-cluster';

{
  apiVersion: 'v1',
  kind: 'Config',
  preferences: {},

  clusters: [
    {
      cluster: {
        server: server,
      },
      name: clusterName,
    },
  ],
  contexts: std.mapWithIndex(function(i, _) {
    context: {
      cluster: clusterName,
      user: 'token-%s' % i,
    },
    name: if i == 0 then clusterName else "%s-%s" % [clusterName, i+1],
  }, tokens),
  'current-context': clusterName,
  users: std.mapWithIndex(function(i, t) {
    name: 'token-%s' % i,
    user: { token: t },
  }, tokens),
}
