#local components = std.extVar("__ksonnet/components");
local components = (import "../components/params.libsonnet");
components + {
  // Insert user-specified overrides here.
}