// get cluster-roles that can use PSPs
MATCH (r:ClusterRole)
WITH apoc.convert.getJsonProperty(r, 'rules') as rules, r
    UNWIND rules as rule
        MATCH (r) WHERE rule.verbs = ["use"] AND rule.resources = ["podsecuritypolicies"] return r