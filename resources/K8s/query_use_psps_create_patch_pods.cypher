// get roles/cluster-roles which use PSPs and can create or patch pods
MATCH (sa:ServiceAccount)-[:ROLE_BINDING]->(r) WHERE r.kind in ["role","clusterrole"]
WITH apoc.convert.getJsonProperty(r, 'rules') as rules, r, sa
UNWIND rules as rule
WITH rule, r, sa
WHERE (("*" IN rule.resources OR "pods" IN rule.resources) AND ("*" IN rule.verbs OR "patch" IN rule.verbs OR "create" in rule.verbs))
MATCH (r)
WITH apoc.convert.getJsonProperty(r, 'rules') as new_rules, r, sa
UNWIND new_rules as new_rule
WITH new_rules, r, sa
WHERE (("podsecuritypolicies" IN new_rule.resources OR "*" IN new_rule.resources) AND ("use" IN new_rule.verbs OR "*" IN new_rule.verbs))
RETURN sa.name