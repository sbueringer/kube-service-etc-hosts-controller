local params = std.extVar("__ksonnet/params").components["kube-service-etc-hosts-operator"];
local k = import "k.libsonnet";
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;
local configMap = k.core.v1.configMap;

local volume = deployment.mixin.spec.template.spec.volumesType;

local targetPort = params.containerPort;
local labels = {app: params.name};
local annotations = {"sidecar.istio.io/inject": "false"};

local caddyConfig = configMap
  .new(
    params.name,
    {Caddyfile: 
     |||
     0.0.0.0:80 {
       markdown
     }
  |||},
  );

local appService = service
  .new(
    params.name,
    labels,
    servicePort.new(params.servicePort, targetPort))
  .withType(params.type);

local appDeployment = deployment
  .new(
    params.name,
    params.replicas,
    container
      .new("operator", params.operatorImage)
      .withEnv(container.envType.new("HOSTS_PATH", "/etcout/hosts"))
      .withVolumeMounts(
        [container.volumeMountsType.new("data", "/data", false),
         container.volumeMountsType.new("etc", "/etcout", false)])
      .withImagePullPolicy("IfNotPresent"),
    labels)
    .withContainersMixin(
      container
      .new("caddy", params.caddyImage)
      .withPorts(containerPort.new(targetPort))
      .withVolumeMounts(
        [container.volumeMountsType.new("data", "/data", true),
        container.volumeMountsType.new("caddy", "/caddy", true)]
      )
      .withImagePullPolicy("IfNotPresent"),
    ) + deployment.mixin.spec.template.spec.withVolumes(
      [volume.fromEmptyDir("data"),
      volume.fromHostPath("etc", "/etc"),
      volume.fromConfigMap("caddy", params.name, {key: "Caddyfile", path: "Caddyfile"})]
    ) + deployment.mixin.metadata.withAnnotations(annotations);

k.core.v1.list.new([appService, appDeployment, caddyConfig])