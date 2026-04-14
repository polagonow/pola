// Package storage implements the storage generator for the Pola CLI.
// It generates StorageBlob and StorageAttachment models using the model
// generator, and configures the storage block in Polafile.hcl.
package storage

import (
	"fmt"

	survey "github.com/AlecAivazis/survey/v2"
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
  pola generate model User name:string avatar:references{StorageBlob}

  # Polymorphic (via StorageAttachment, any model can have attachments)
  # StorageAttachment already has record:references{polymorphic}`,
		Args: cobra.NoArgs,
		RunE: g.run,
		Example: `  pola generate storage
  pola generate storage --driver rclone --root myremote:bucket/path
  pola generate storage --driver rclone --root myremote:bucket/path --config-path /etc/rclone/rclone.conf
  pola generate storage --driver fs --root ./uploads`,
	}
	cmd.Flags().String("driver", "fs", "storage driver: fs or rclone")
	cmd.Flags().String("root", "uploads", "storage root path (local dir for fs, remote:path for rclone)")
	cmd.Flags().String("config-path", "", "path to rclone config file (rclone driver)")
	return cmd
}

func (g *StorageGenerator) run(cmd *cobra.Command, _ []string) error {
	projectDir, err := project.FindRoot()
	if err != nil {
		return err
	}

	driver, _ := cmd.Flags().GetString("driver")
	root, _ := cmd.Flags().GetString("root")
	configPath, _ := cmd.Flags().GetString("config-path")

	// --- Update Polafile with storage config ---

	pf, err := polafile.Load(projectDir)
	if err != nil {
		return fmt.Errorf("load Polafile: %w", err)
	}
	if pf == nil {
		return fmt.Errorf("no Polafile.hcl found; run 'pola new' first or create one manually")
	}

	pf.Storage = &polafile.StorageConfig{
		Driver:     driver,
		Root:       root,
		ConfigPath: configPath,
	}

	// Ensure database block exists; prompt interactively for missing ORM/adapter.
	if pf.Database == nil {
		pf.Database = &polafile.Database{
			Models: "db/models",
			Migrations: &polafile.Migrations{
				Directory: "db/migrations",
				Format:    "hcl",
			},
			Envs: []polafile.DatabaseEnvironment{
				{Environment: "development", Adapter: "sqlite"},
				{Environment: "production", Adapter: "postgresql"},
			},
		}
	}
	if pf.Database.ORM == "" {
		orm, err := promptSelect("ORM:", []string{"beego", "ent", "gorm"})
		if err != nil {
			return err
		}
		pf.Database.ORM = orm
	}
	if pf.Database.Adapter == "" {
		adapter, err := promptSelect("Database adapter:", []string{"postgresql", "mysql", "sqlite"})
		if err != nil {
			return err
		}
		pf.Database.Adapter = adapter
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
		"owner:references{polymorphic}",
		"storage_blob_id:int:index",
	}
	if err := generators.Run("model", cmd, attachmentArgs); err != nil {
		return fmt.Errorf("model StorageAttachment: %w", err)
	}

	fmt.Println("\nStorage setup complete!")
	fmt.Println("You can now associate models with files:")
	fmt.Println("  pola generate model User name:string avatar:references{StorageBlob}  # direct FK to StorageBlob")
	fmt.Println("  # Or use StorageAttachment for polymorphic attachments")

	return nil
}

func promptSelect(message string, options []string) (string, error) {
	var answer string
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}
	if err := survey.AskOne(prompt, &answer); err != nil {
		return "", fmt.Errorf("prompt: %w", err)
	}
	return answer, nil
}
