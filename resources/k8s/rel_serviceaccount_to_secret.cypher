MATCH (s:Secret)
WITH s
MATCH (x {uid: s.`metadata.annotations.kubernetes.io/service-account.uid`})
MERGE (x)-[:SERVICE_ACCOUNT_TOKEN]->(s)
RETURN *