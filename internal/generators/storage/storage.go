// Package storage implements the storage generator for the Pola CLI.
// It generates StorageBlob and StorageAttachment models using the model
// generator, and configures the storage block in Polafile.hcl.
package storage

import (
	"fmt"

	"github.com/polagonow/pola/internal/generators"
	"github.com/polagonow/pola/internal/project"
	"github.com/polagonow/pola/polafile"
	"github.com/spf13/cobra"
)

// StorageGenerator scaffolds file storage infrastructure.
type StorageGenerator struct{}

func init() {
	generators.Register(&StorageGenerator{})
}

func (g *StorageGenerator) Name() string        { return "storage" }
func (g *StorageGenerator) Description() string  { return "Set up file storage models and configuration" }
func (g *StorageGenerator) AfterHooks() []generators.Hook { return nil }

func (g *StorageGenerator) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Set up file storage models and configuration",
		Long: `Generate StorageBlob and StorageAttachment models for file storage,
and configure the storage block in Polafile.hcl.

StorageBlob stores file metadata (key, filename, content type, size, checksum).
StorageAttachment is a polymorphic join table linking any model to blobs.

After running this, you can associate files with models:

  # Direct relationship (User has one avatar blob)
  pola generate model User name:string avatar:references

  # Polymorphic (via StorageAttachment, any model can have attachments)
  # StorageAttachment already has record:references{polymorphic}`,
		Args: cobra.NoArgs,
		RunE: g.run,
		Example: `  pola generate storage
  pola generate storage --driver s3 --bucket my-uploads
  pola generate storage --driver fs --root ./uploads`,
	}
	cmd.Flags().String("driver", "fs", "storage driver: fs or s3")
	cmd.Flags().String("root", "uploads", "local root directory (fs driver)")
	cmd.Flags().String("remote", "s3", "rclone remote name (s3 driver)")
	cmd.Flags().String("bucket", "", "rclone bucket or path (s3 driver)")
	return cmd
}

func (g *StorageGenerator) run(cmd *cobra.Command, _ []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	driver, _ := cmd.Flags().GetString("driver")
	root, _ := cmd.Flags().GetString("root")
	remote, _ := cmd.Flags().GetString("remote")
	bucket, _ := cmd.Flags().GetString("bucket")

	// --- Update Polafile with storage config ---

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return fmt.Errorf("no Polafile.hcl found; run 'pola new' first or create one manually")
	}

	pf.Storage = &polafile.StorageConfig{
		Driver: driver,
	}
	switch driver {
	case "fs":
		pf.Storage.Root = root
	case "s3":
		pf.Storage.Remote = remote
		pf.Storage.Bucket = bucket
	}

	if err := polafile.Save(projectDir, pf); err != nil {
		return fmt.Errorf("save Polafile: %w", err)
	}
	fmt.Println("Updated Polafile.hcl with storage configuration")

	// --- Generate StorageBlob model ---

	fmt.Println("Generating StorageBlob model...")
	blobArgs := []string{
		"StorageBlob",
		"key:string:uniq",
		"filename:string",
		"content_type:string",
		"byte_size:int64",
		"checksum:string",
	}
	if err := generators.Run("model", cmd, blobArgs); err != nil {
		return fmt.Errorf("model StorageBlob: %w", err)
	}

	// --- Generate StorageAttachment model ---

	fmt.Println("Generating StorageAttachment model...")
	attachmentArgs := []string{
		"StorageAttachment",
		"name:string:index",
		"record:references{polymorphic}",
		"blob:references",
	}
	if err := generators.Run("model", cmd, attachmentArgs); err != nil {
		return fmt.Errorf("model StorageAttachment: %w", err)
	}

	fmt.Println("\nStorage setup complete!")
	fmt.Println("You can now associate models with files:")
	fmt.Println("  pola generate model User name:string avatar:references  # direct FK to StorageBlob")
	fmt.Println("  # Or use StorageAttachment for polymorphic attachments")

	return nil
}
