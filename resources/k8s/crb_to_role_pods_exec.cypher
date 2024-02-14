MATCH (sa:ServiceAccount)-[rb:ROLE_BINDING]->(r) WHERE r.kind in ["clusterrole","role"]
WITH apoc.convert.getJsonProperty(r, 'rules') as rules,r,sa,rb
UNWIND rules as rule
WITH rule,r,sa,rb
WHERE (("*" IN rule.resources OR "pods/exec" IN rule.resources) AND ("*" IN rule.verbs OR "create" in rule.verbs OR "get" in rule.verbs) AND NOT ("kube-system" IN sa.`metadata.namespace`))
MATCH (r)
RETURN DISTINCT(sa.name), sa.`metadata.namespace`, r.name