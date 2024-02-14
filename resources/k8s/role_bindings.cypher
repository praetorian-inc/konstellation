CALL apoc.load.json('{{ path }}', '$.items[*]') YIELD value as rbs
WITH rbs.metadata.name as rbname, rbs.kind as rbkind, rbs.roleRef.name as rname, rbs.roleRef.kind as rkind, rbs.subjects as subjects
    UNWIND subjects as s
        WITH *, s.kind as kind, apoc.text.split(s.name, ":") as user
        // run a node merge to prime 
        CALL apoc.merge.node([s.kind], {name: s.name, kind: toLower(s.kind)}, {`spec.group`: s.apiGroup}) YIELD node
        WITH *
        CALL apoc.do.case(
        [
            // system:serviceaccount:foo:admin-deployer
            kind = "User" AND user[3] IS NOT NULL,
            'MATCH (subject:ServiceAccount {name: user[3], namespace: user[2]}) RETURN subject',

            // system:kube-proxy
            kind = "User" and size(user) = 2,
            'MATCH (subject:ServiceAccount {name: user[1]}) RETURN subject'
        ],
            'MATCH (subject {name: name}) WHERE subject.kind = toLower(kind) return subject',
            {kind: toLower(s.kind), name: s.name, user: apoc.text.split(s.name, ":"), apiGroup: s.apiGroup}
        ) YIELD value as subject
        WITH subject.subject as subject, rkind, rname, rbname, rbkind

        //MATCH (s {kind: subject.kind, name: subject.name, namespace: subject.namespace}) WHERE id(s) = subject.identity
        //CALL apoc.nodes.get(ID(subject)) YIELD node as s
        
        WITH *
        CALL apoc.merge.node([rkind], {name: rname}) YIELD node as role
        
        WITH *
        MERGE (subject)-[r:ROLE_BINDING {name: rbname, kind: rbkind}]->(role)
        RETURN subject, TYPE(r), r.name, role.name