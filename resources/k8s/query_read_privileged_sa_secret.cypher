MATCH (y)-[z:ROLE_BINDING]->(x)-[a:GET|LIST]->(s:Secret)<-[b:SERVICE_ACCOUNT_TOKEN]-(sa:ServiceAccount)-[r:ROLE_BINDING]->(role:ClusterRole)
WHERE x.kind in ["clusterrole", "role"] and not y.name = sa.name and role.name in ["admin", "cluster-admin"]
RETURN role.name as ClusterRole, sa.name as ServiceAccount, s.name as Secret, collect(distinct y.name) as readers