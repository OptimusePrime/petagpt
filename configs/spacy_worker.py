from time import sleep

import spacy, sys, json

nlp = spacy.load("hr_core_news_lg")


def handle(method, data):
    if method == "senter":
        text = data.get("text", "")
        doc = nlp(text)

        out = {
            "sentences": list(map(str, list(doc.sents)))
        }
        return out
    return None


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        print(type(line))
        req = json.loads(line)
        try:
            result = handle(req["method"], req["data"])
            out = {"id": req["id"], "result": result}
        except Exception as e:
            print(e)
            out = {"id": req.get("id", ""), "error": str(e)}
        sys.stdout.write(json.dumps(out) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
