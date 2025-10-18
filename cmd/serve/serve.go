package serve

import (
	"context"
	"fmt"

	"github.com/OptimusePrime/petagpt/internal/parser"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the PetaGPT server",
	RunE: func(cmd *cobra.Command, args []string) error {
		//port := viper.GetInt("port")
		//
		//fmt.Printf("Starting PetaGPT server on port %d...\n", port)
		ctx := context.Background()

		chunker, err := parser.NewDocumentChunker(cmd.Context(), 8)
		if err != nil {
			return err
		}
		fmt.Println("started chunker")

		sentences, err := chunker.SentenceSegmentText(ctx, "Pozdrav svima! Ja sam Karlo.")
		if err != nil {
			return err
		}
		fmt.Println(sentences)

		/*		err = w.Call(ctx, parser.SENTENCE_SEGMENTATION, In{Text: "Pozdrav svima! Ja sam Karlo."}, &out)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(out.Sentences)
				err = w.Call(ctx, parser.SENTENCE_SEGMENTATION, In{Text: "Bok svima! Ja sam Jovan."}, &out)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(out.Sentences)*/

		return nil
	},
}

func NewCommand() *cobra.Command {
	return serveCmd
}
