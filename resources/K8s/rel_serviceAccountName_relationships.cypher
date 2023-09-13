MATCH (node1)
WITH node1
// ServiceAccounts are namespaced, constrain on match
MATCH (node2:ServiceAccount {`metadata.namespace`: node1.`metadata.namespace`}) 
WHERE node1.`spec.serviceAccountName` IS NOT NULL AND node1.`spec.serviceAccountName`= node2.name OR
node1.`spec.template.spec.serviceAccountName` IS NOT NULL AND node1.`spec.template.spec.serviceAccountName`= node2.name OR
node1.`spec.podSpec.serviceAccountName` IS NOT NULL AND node1.`spec.podSpec.serviceAccountName`= node2.name OR
node1.`spec.jobTemplate.spec.template.spec.serviceAccountName` IS NOT NULL AND node1.`spec.jobTemplate.spec.template.spec.serviceAccountName` = node2.name
MERGE (node1)-[r:RUN_AS]->(node2)
RETURN node1.name, node2.name, node1.`metadata.namespace`