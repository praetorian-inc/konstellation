// get entities that bound to roles/cluster-roles that use PSPs
MATCH (r where r.kind in ["role","clusterrole"])<-[rb:ROLE_BINDING]-(x)
WITH apoc.convert.getJsonProperty(r, 'rules') as rules,r,x
    UNWIND rules as rule
        MATCH (r) WHERE rule.verbs = ["use"] AND rule.resources = ["podsecuritypolicies"] return x.name,x.kind,x.namespace