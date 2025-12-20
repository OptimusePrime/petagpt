package main

import (
	_ "embed"

	"github.com/OptimusePrime/petagpt/cmd"
	// "github.com/OptimusePrime/petagpt/cmd"
)

func main() {
	//index.TestBleve()
	cmd.Execute()
	// configs.InitConfig(nil)
	// msgS, err := safety.CheckAssistantMessageSafety(context.Background(), "Jebem ti mater")
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// fmt.Println(msgS.SafetyLevel)
	// fmt.Println(msgS.SafetyCategory)
	//result, err := index.SearchChromaCollection(context.Background(), "vgim", 100, "Primijenejna informatika")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//fmt.Println(result.GetDocumentsGroups()[0][0].ContentString())
	//result, err := index.SearchBleveIndex("/home/optimuseprime/.config/.petagpt/bm25/vgim.bleve", "informatika", 100)
	//if err != nil {
	//	log.Fatalln(err)
	//}

	//result, err := index.SearchIndex(context.Background(), "vgim", "informatika", 20)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//
	//fmt.Println(result.Documents)
}
