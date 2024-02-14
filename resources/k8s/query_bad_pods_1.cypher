// Based on https://bishopfox.com/blog/kubernetes-pod-privilege-escalation Bad Pod #1
MATCH (p:Pod {`spec.hostPID`: "true", `spec.hostIPC`: "true", `spec.hostNetwork`: "true"}) 
WITH apoc.convert.getJsonProperty(p, 'spec.volumes') as volumes, p
    UNWIND volumes as volume
        MATCH (p) WHERE volume.hostPath IS NOT NULL
        WITH p, volume
        MATCH (p)-[:POD]->(c:Container {`securityContext.privileged`: "true"})
        WITH apoc.convert.getJsonProperty(c, 'volumeMounts') as volumeMounts, p, c, volume
        UNWIND volumeMounts as volumeMount
            MATCH (p)-[:POD]->(c:Container) WHERE volume.name = volumeMount.name
            RETURN p.name, volume.name, volume.hostPath.path, c.name, volumeMount