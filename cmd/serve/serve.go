package serve

import (
	"github.com/OptimusePrime/petagpt/internal/server"
	"github.com/spf13/cobra"
)

var port string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the PetaGPT server",
	RunE: func(cmd *cobra.Command, args []string) error {
		//ctx := context.Background()
		//
		////sem := semaphore.NewWeighted(viper.GetInt64("context_llm.max_concurrent_requests"))
		//chunker, err := parser.NewDocumentChunker(cmd.Context(), 8, 2)
		//if err != nil {
		//	return err
		//}
		//
		//path := "/home/optimuseprime/Downloads/Plan_za_Primijenjenu_informatiku-4.pdf"
		//
		//file, err := os.ReadFile(path)
		//if err != nil {
		//	return err
		//}
		//
		//chunks, err := parser.ProcessDocument(ctx, file, filepath.Ext(path), chunker, 50)
		//if err != nil {
		//	return err
		//}
		//for i, s := range chunks {
		//	fmt.Printf("%d: %s\n", i, s.String())
		//}

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

		server.StartServer("vgim", 20)

		return nil
	},
}

func NewCommand() *cobra.Command {
	return serveCmd
}
