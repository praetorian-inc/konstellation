// service accounts outside of `kube-system` with clusterrolebindings or rolebindings to roles or clusterroles that allow them full control or create/patch access to pods
MATCH (sa:ServiceAccount)-[rb:ROLE_BINDING]->(r) WHERE r.kind in ["clusterrole","role"]
WITH apoc.convert.getJsonProperty(r, 'rules') as rules, r,sa,rb
UNWIND rules as rule
WITH rule, r,sa,rb
WHERE (("*" IN rule.resources OR "pods" IN rule.resources) AND ("*" IN rule.verbs OR "create" in rule.verbs OR "patch" in rule.verbs) AND NOT ("kube-system" IN sa.`metadata.namespace`))
MATCH (r)
WITH apoc.convert.getJsonProperty(r, 'rules') as new_rules,r,sa,rb
UNWIND new_rules as new_rule
WITH new_rules,r,sa,rb
WHERE (("podsecuritypolicies" IN new_rule.resources OR "*" IN new_rule.resources) AND ("use" IN new_rule.verbs OR "*" IN new_rule.verbs))
RETURN DISTINCT(sa.name), sa.`metadata.namespace`, sa.kind, r.name, rb.name, rb.kind, rb.`metadata.namespace`