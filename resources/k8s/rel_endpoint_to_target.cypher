MATCH (e:Endpoints)
WITH apoc.convert.getJsonProperty(e, 'subsets') as subsets, e
UNWIND subsets as subset
    UNWIND subset.addresses as address
    MATCH (x) WHERE x.`metadata.uid` = address.targetRef.uid and x.`metadata.uid` IS NOT NULL
    MERGE (e)-[:TARGET]->(x)
    RETURN *