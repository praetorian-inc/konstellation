match (z) WHERE z.`aggregationRule.clusterRoleSelectors` IS NOT NULL
WITH apoc.convert.getJsonProperty(z, 'aggregationRule.clusterRoleSelectors') as agg, z
UNWIND agg as rule
    WITH z, keys(rule.matchLabels) as keys
    UNWIND keys as k
    CALL apoc.cypher.run('MATCH (x {`metadata.labels.' + k + '`: "true"}) RETURN x', {}) YIELD value
    WITH z, value.x as x
    MATCH (x)-[r1]-(y)
    WITH *, collect(r1) as relationships, collect(y) as nodes
    UNWIND relationships as r
        CALL apoc.create.relationship(z, type(r), {aggregated: true, aggregationRule: k + ':' +}, endNode(r)) YIELD rel
        RETURN rel
        