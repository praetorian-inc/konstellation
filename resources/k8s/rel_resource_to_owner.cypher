MATCH (r) WHERE r.`metadata.ownerReferences` IS NOT NULL
WITH apoc.convert.getJsonProperty(r, 'metadata.ownerReferences') as owners, r
UNWIND owners as owner
    WITH *
    MATCH (x {uid: owner.uid})
    MERGE (x)-[:OWNER]->(r)
    RETURN *