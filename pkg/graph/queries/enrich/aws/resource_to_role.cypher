MATCH (res:Resource) 
WHERE NOT res.Role IS NULL 
WITH res
MATCH (p:Principal) WHERE p.arn = res.Role
MERGE (res)-[r:HAS_ROLE]->(p)
RETURN res,r,p