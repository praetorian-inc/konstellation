// get entities (and their roles) that have exec permissions into pods within their namespace/cluster
MATCH (x)-[rb:ROLE_BINDING]->(r where r.kind in ["role","clusterrole"]) with apoc.convert.getJsonProperty(r,'rules') as rules,r,x
    unwind rules as rule
        match (r) where ("create" in rule.verbs or "*" in rule.verbs) and ("pods/exec" in rule.resources or "*" in rule.resources)
        RETURN r.name,r.kind,collect(DISTINCT x.name),r.namespace