package app

import (
	"context"
	"fmt"

	"github.com/formation-res/open-location-hub-cli/internal/cli"
	"github.com/formation-res/open-location-hub-cli/internal/output"
	"github.com/spf13/cobra"
)

type resourceSpec struct {
	Name      string
	Singular  string
	ReadArg   string
	WriteArg  string
	Example   string
	Summary   func(context.Context, *cli.Config) (any, error)
	DeleteAll func(context.Context, *cli.Config) error
	List      func(context.Context, *cli.Config) (any, error)
	Get       func(context.Context, *cli.Config, string) (any, error)
	Create    func(context.Context, *cli.Config, string) (any, error)
	Update    func(context.Context, *cli.Config, string, string) (any, error)
	Delete    func(context.Context, *cli.Config, string) error
}

func newResourceCommand(cfg *cli.Config, printer *output.Printer, spec resourceSpec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   spec.Name,
		Short: fmt.Sprintf("Manage %s", spec.Name),
		Long:  fmt.Sprintf("CRUD operations for %s.\n\nCreate and update accept JSON or YAML via --file.", spec.Name),
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s", spec.Name),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := spec.List(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			return printer.Print(v)
		},
	})

	if spec.Summary != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "summary",
			Short: fmt.Sprintf("Get a summary of %s", spec.Name),
			RunE: func(cmd *cobra.Command, args []string) error {
				v, err := spec.Summary(cmd.Context(), cfg)
				if err != nil {
					return err
				}
				return printer.Print(v)
			},
		})
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "get " + spec.ReadArg,
		Short: fmt.Sprintf("Get a %s", spec.Singular),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			v, err := spec.Get(cmd.Context(), cfg, args[0])
			if err != nil {
				return err
			}
			return printer.Print(v)
		},
	})

	create := &cobra.Command{
		Use:   "create --file payload.json",
		Short: fmt.Sprintf("Create a %s", spec.Singular),
		Long:  spec.Example,
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			v, err := spec.Create(cmd.Context(), cfg, file)
			if err != nil {
				return err
			}
			return printer.Print(v)
		},
	}
	create.Flags().StringP("file", "f", "", "Read request body from file or - for stdin")
	must(create.MarkFlagRequired("file"))
	cmd.AddCommand(create)

	update := &cobra.Command{
		Use:   "update " + spec.WriteArg + " --file payload.json",
		Short: fmt.Sprintf("Update a %s", spec.Singular),
		Long:  spec.Example,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			v, err := spec.Update(cmd.Context(), cfg, args[0], file)
			if err != nil {
				return err
			}
			return printer.Print(v)
		},
	}
	update.Flags().StringP("file", "f", "", "Read request body from file or - for stdin")
	must(update.MarkFlagRequired("file"))
	cmd.AddCommand(update)

	cmd.AddCommand(&cobra.Command{
		Use:   "delete " + spec.ReadArg,
		Short: fmt.Sprintf("Delete a %s", spec.Singular),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := spec.Delete(cmd.Context(), cfg, args[0]); err != nil {
				return err
			}
			printer.Success("%s deleted: %s", spec.Singular, args[0])
			return nil
		},
	})

	if spec.DeleteAll != nil {
		cmd.AddCommand(&cobra.Command{
			Use:   "delete-all",
			Short: fmt.Sprintf("Delete all %s", spec.Name),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := spec.DeleteAll(cmd.Context(), cfg); err != nil {
					return err
				}
				printer.Success("%s deleted", spec.Name)
				return nil
			},
		})
	}

	return cmd
}
