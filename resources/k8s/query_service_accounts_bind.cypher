// find service accounts that can use the "bind" verb (https://raesene.github.io/blog/2021/01/16/Getting-Into-A-Bind-with-Kubernetes/)
MATCH (sa:ServiceAccount)-[rb:ROLE_BINDING]->(r) WHERE r.kind in ["clusterrole","role"]
WITH apoc.convert.getJsonProperty(r, 'rules') as rules,r,sa,rb
UNWIND rules as rule
WITH rule,r,sa,rb
WHERE ("bind" in rule.verbs) AND NOT ("kube-system" IN sa.`metadata.namespace`)
MATCH (r)
RETURN DISTINCT(sa.name), sa.`metadata.namespace`, r.name, rb.name, rb.kind, rb.`metadata.namespace`