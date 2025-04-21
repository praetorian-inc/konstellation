// Find roles that don't have AdministratorAccess but can assume roles that do
MATCH (source:Role)-[r:`sts:AssumeRole`]->(target:Role)
WHERE 
    // Target role has AdministratorAccess
    (target.attachedManagedPolicies IS NOT NULL AND 
     ANY(policy IN target.attachedManagedPolicies WHERE policy CONTAINS 'AdministratorAccess'))
    
    // Source role doesn't have AdministratorAccess
    AND (source.attachedManagedPolicies IS NULL OR 
         NOT ANY(policy IN source.attachedManagedPolicies WHERE policy CONTAINS 'AdministratorAccess'))
    
    // Exclude AWS reserved or service roles
    AND NOT (source.roleName STARTS WITH 'AWSReservedSSO_')
    
    // Filter out self-references
    AND source.roleName <> target.roleName

WITH *
RETURN *