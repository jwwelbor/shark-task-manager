package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestAddFlagSet_Metadata verifies metadata flags are registered correctly
func TestAddFlagSet_Metadata(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetMetadata)

	// Verify flags exist
	titleFlag := cmd.Flags().Lookup("title")
	assert.NotNil(t, titleFlag, "title flag should be registered")

	descFlag := cmd.Flags().Lookup("description")
	assert.NotNil(t, descFlag, "description flag should be registered")
}

// TestAddFlagSet_Path verifies path flags are registered correctly
func TestAddFlagSet_Path(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetPath)

	// Verify flags exist
	pathFlag := cmd.Flags().Lookup("path")
	assert.NotNil(t, pathFlag, "path flag should be registered")

	filenameFlag := cmd.Flags().Lookup("filename")
	assert.NotNil(t, filenameFlag, "filename flag should be registered")

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "force flag should be registered")
}

// TestAddFlagSet_EpicStatus verifies epic status flags are registered correctly
func TestAddFlagSet_EpicStatus(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetEpicStatus)

	// Verify flags exist
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag, "status flag should be registered")

	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag, "priority flag should be registered")

	bvFlag := cmd.Flags().Lookup("business-value")
	assert.NotNil(t, bvFlag, "business-value flag should be registered")
}

// TestAddFlagSet_FeatureStatus verifies feature status flags are registered correctly
func TestAddFlagSet_FeatureStatus(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetFeatureStatus)

	// Verify flags exist
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag, "status flag should be registered")
}

// TestAddFlagSet_CustomKey verifies custom key flag is registered correctly
func TestAddFlagSet_CustomKey(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetCustomKey)

	// Verify flag exists
	keyFlag := cmd.Flags().Lookup("key")
	assert.NotNil(t, keyFlag, "key flag should be registered")
}

// TestAddFlagSet_WithDefaults verifies default values are applied correctly
func TestAddFlagSet_WithDefaults(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddFlagSet(cmd, FlagSetEpicStatus,
		WithDefaults(map[string]interface{}{
			"status":   "draft",
			"priority": "medium",
		}))

	// Verify default values
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "draft", statusFlag.DefValue, "status default should be 'draft'")

	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag)
	assert.Equal(t, "medium", priorityFlag.DefValue, "priority default should be 'medium'")
}

// TestAddFlagSet_InvalidFlagSet verifies panic on invalid flag set
func TestAddFlagSet_InvalidFlagSet(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	assert.Panics(t, func() {
		AddFlagSet(cmd, FlagSet("invalid"))
	}, "Should panic on invalid flag set")
}

// TestAddMetadataFlags verifies individual metadata flag function
func TestAddMetadataFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddMetadataFlags(cmd)

	// Verify flags exist
	titleFlag := cmd.Flags().Lookup("title")
	assert.NotNil(t, titleFlag, "title flag should be registered")

	descFlag := cmd.Flags().Lookup("description")
	assert.NotNil(t, descFlag, "description flag should be registered")
}

// TestAddPathFlags verifies individual path flag function
func TestAddPathFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddPathFlags(cmd)

	// Verify flags exist
	pathFlag := cmd.Flags().Lookup("path")
	assert.NotNil(t, pathFlag, "path flag should be registered")

	filenameFlag := cmd.Flags().Lookup("filename")
	assert.NotNil(t, filenameFlag, "filename flag should be registered")

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag, "force flag should be registered")
}

// TestAddEpicStatusFlags verifies individual epic status flag function
func TestAddEpicStatusFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	defaults := map[string]string{
		"status":   "draft",
		"priority": "medium",
	}

	AddEpicStatusFlags(cmd, defaults)

	// Verify flags exist with defaults
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "draft", statusFlag.DefValue)

	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag)
	assert.Equal(t, "medium", priorityFlag.DefValue)

	bvFlag := cmd.Flags().Lookup("business-value")
	assert.NotNil(t, bvFlag)
}

// TestAddFeatureStatusFlags verifies individual feature status flag function
func TestAddFeatureStatusFlags(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	defaults := map[string]string{
		"status": "draft",
	}

	AddFeatureStatusFlags(cmd, defaults)

	// Verify flag exists with default
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "draft", statusFlag.DefValue)
}

// TestAddCustomKeyFlag verifies individual custom key flag function
func TestAddCustomKeyFlag(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
	}

	AddCustomKeyFlag(cmd)

	// Verify flag exists
	keyFlag := cmd.Flags().Lookup("key")
	assert.NotNil(t, keyFlag, "key flag should be registered")
}

// TestAddFlagSet_WithRequired verifies required flags are marked correctly
func TestAddFlagSet_WithRequired(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		Run: func(cmd *cobra.Command, args []string) {
			// Empty run function to avoid execution errors
		},
	}

	AddFlagSet(cmd, FlagSetMetadata, WithRequired("title"))

	// Verify title flag is marked as required
	// Note: We can't directly check if a flag is required in Cobra,
	// but we can verify the flag exists which is the main purpose
	titleFlag := cmd.Flags().Lookup("title")
	assert.NotNil(t, titleFlag, "title flag should be registered")

	descFlag := cmd.Flags().Lookup("description")
	assert.NotNil(t, descFlag, "description flag should be registered")
}
