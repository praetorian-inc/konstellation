package transformers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/praetorian-inc/konstellation/pkg/graph"
	"github.com/praetorian-inc/konstellation/pkg/graph/utils"
	"github.com/praetorian-inc/nebula/pkg/types"
)

func NodeFromEnrichedResourceDescription(erd *types.EnrichedResourceDescription) (*graph.Node, error) {
	typeSplit := strings.Split(erd.TypeName, "::")
	props, err := utils.ConvertAndFlatten(erd.Properties)
	if err != nil {
		slog.Error("failed to convert and flatten properties", "error", err)
		props = make(map[string]interface{})
	}

	props["platform"] = "aws"
	props["service"] = erd.Service()
	props["arn"] = erd.Arn.String()
	props["region"] = erd.Region
	props["account"] = erd.AccountId
	props["tags"] = stringify(erd.Tags())
	props["type"] = erd.TypeName

	// Check if this is a role resource and add Principal label
	labels := []string{
		erd.TypeName,                      // e.g. "AWS::IAM::Role"
		strings.Join(typeSplit[1:], "::"), // e.g. "IAM::Role"
		"Resource",
	}

	// Add the Principal label to roles to ensure consistency
	if erd.TypeName == "AWS::IAM::Role" {
		labels = append(labels, "Principal", "Role")
	}

	node := graph.Node{
		Labels:     labels,
		Properties: props,
		UniqueKey:  []string{"arn"},
	}
	return &node, nil
}

func NodeFromUserDL(user *types.UserDL) *graph.Node {

	return &graph.Node{
		Labels: []string{"User", "Principal"},
		Properties: map[string]interface{}{
			"platform":                "aws",
			"arn":                     user.Arn,
			"userId":                  user.UserId,
			"userName":                user.UserName,
			"path":                    user.Path,
			"createDate":              user.CreateDate,
			"groupList":               user.GroupList,
			"attachedManagedPolicies": getAttachedManagedPoliciesList(user.AttachedManagedPolicies),
			"userPolicyList":          stringify(user.UserPolicyList),
			"permissionsBoundary":     stringify(user.PermissionsBoundary),
			"tags":                    stringify(user.Tags),
		},
		UniqueKey: []string{"arn"},
	}
}

func NodeFromRoleDL(role *types.RoleDL) *graph.Node {

	return &graph.Node{
		Labels: []string{"Role", "Principal"},
		Properties: map[string]interface{}{
			"platform":                "aws",
			"arn":                     role.Arn,
			"roleName":                role.RoleName,
			"roleId":                  role.RoleId,
			"path":                    role.Path,
			"createDate":              role.CreateDate,
			"assumeRolePolicyDoc":     stringify(role.AssumeRolePolicyDocument),
			"tags":                    stringify(role.Tags),
			"rolePolicyList":          stringify(role.RolePolicyList),
			"permissionsBoundary":     stringify(role.PermissionsBoundary),
			"instanceProfileList":     stringify(role.InstanceProfileList),
			"attachedManagedPolicies": getAttachedManagedPoliciesList(role.AttachedManagedPolicies),
		},
		UniqueKey: []string{"arn"},
	}
}

func NodeFromGroupDL(group *types.GroupDL) *graph.Node {

	return &graph.Node{
		Labels: []string{"Group", "Principal"},
		Properties: map[string]interface{}{
			"platform":                "aws",
			"groupName":               group.GroupName,
			"groupId":                 group.GroupId,
			"arn":                     group.Arn,
			"createDate":              group.CreateDate,
			"groupPolicyList":         stringify(group.GroupPolicyList),
			"attachedManagedPolicies": getAttachedManagedPoliciesList(group.AttachedManagedPolicies),
			"path":                    group.Path,
		},
		UniqueKey: []string{"arn"},
	}
}

func stringify(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(jsonData)
}

func getAttachedManagedPoliciesList(mpl []types.ManagedPL) []string {
	var amp []string
	for _, pol := range mpl {
		amp = append(amp, pol.PolicyArn)
	}
	return amp
}

func processResourceProperties(erd types.EnrichedResourceDescription) (map[string]interface{}, error) {
	// First, try to handle the Properties field as potentially escaped JSON
	if str, ok := erd.Properties.(string); ok {
		// Try to unescape it
		unescaped, err := utils.UnescapeJSONString(str)
		if err == nil {
			// Parse the unescaped JSON
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(unescaped), &data); err == nil {
				return utils.FlattenJSON(data), nil
			}
		}
	}

	// Fall back to the standard ConvertAndFlatten
	return utils.ConvertAndFlatten(erd.Properties)
}
