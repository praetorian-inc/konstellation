MATCH (n)-[:FULL_CONTROL]->(r:ClusterRole)
WHERE r.name =~ "^system:.*$" and not n.name contains "system" and not n.name = "cluster-admin"
RETURN DISTINCT n.name