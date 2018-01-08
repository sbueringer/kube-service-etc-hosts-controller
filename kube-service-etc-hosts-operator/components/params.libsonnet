{
  global: {
    // User-defined global parameters; accessible to all component and environments, Ex:
    // replicas: 4,
  },
  components: {
    // Component-level parameters, defined initially from 'ks prototype use ...'
    // Each object below should correspond to a component in the components/ directory
    "kube-service-etc-hosts-operator": {
      containerPort: 80,
      operatorImage: "docker.io/sbueringer/kube-service-etc-hosts-operator",
      caddyImage: "docker.io/sbueringer/caddy",
      name: "kube-service-etc-hosts-operator",
      replicas: 1,
      servicePort: 80,
      type: "ClusterIP",
    },
  },
}
