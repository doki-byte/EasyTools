import Tippy from "@tippyjs/react";
import { CompositeDecorator, ContentState, Editor, EditorState } from "draft-js";
import { useEffect, useRef, useState } from "react";
import { ENVIRONMENT_REGEX, extractEnv } from "../../utils/utils";

const DraftEditor = ({ value, setValue, fontsm = true, envVars, invoke = null }) => {
  const [focus, setfocus] = useState(false);
  const drafteditRef = useRef(null);
  const [editorState, setEditorState] = useState(EditorState.createWithContent(ContentState.createFromText(value), getDecorators()));
  const onHandleChanage = (e) => {
    setEditorState(e);
    const text = e.getCurrentContent().getPlainText();
    setValue(text);
  };
  useEffect(() => {
    const newDecorator = getDecorators(envVars);
    const newEditorState = EditorState.set(editorState, {
      decorator: newDecorator,
    });
    setEditorState(newEditorState);
  }, [envVars]);

  const onHandleReturn = () => {
    if (invoke) {
      invoke();
    }
    return "handled";
  };

  return (
    <div
      className={`h-full break-all w-full text-lit flex items-center cursor-text ${fontsm ? "text-sm" : "text-lg"}`}
      onClick={() => drafteditRef.current.focus()}
    >
      <div
        className={`${focus ? "h-fit min-h-full relative self-start z-50 text-txtlit outline-2 outline-blue-500 pt-1" : `${fontsm ? "h-5 text-txtprim" : "h-8"} overflow-hidden`} w-full bg-brand rounded-sm break-all px-2`}
      >
        <Editor
          ref={drafteditRef}
          editorState={editorState}
          onChange={(e) => onHandleChanage(e)}
          autoCapitalize="false"
          handleReturn={() => onHandleReturn()}
          onFocus={() => setfocus(true)}
          onBlur={() => setfocus(false)}
          stripPastedStyles={true}
        />
      </div>
    </div>
  );
};
function getDecorators(envVars) {
  return new CompositeDecorator([
    {
      strategy: hashtagStrategy,
      component: HandleSpan,
      props: { envVars },
    },
  ]);
}
function hashtagStrategy(contentBlock, callback) {
  findWithRegex(ENVIRONMENT_REGEX, contentBlock, callback);
}

function findWithRegex(regex, contentBlock, callback) {
  const text = contentBlock.getText();
  let matchArr, start;
  while ((matchArr = regex.exec(text)) !== null) {
    start = matchArr.index;
    callback(start, start + matchArr[0].length);
  }
}
const HandleSpan = (props) => {
  let clx = "!bg-gray-600";
  let h = "Value: Variable Not found!";
  let output = extractEnv(props?.decoratedText);
  if (output) {
    let y = props?.envVars && props.envVars[output];
    if (y) {
      clx = "!text-green-600";
      h = `Value: ${y}`;
    } else {
      clx = "!text-red-600";
    }
  }
  return (
    <Tippy
      content={
        <p className="truncate whitespace-nowrap overflow-ellipsis" style={{ maxWidth: "300px", maxHeight: "300px" }}>
          {h}
        </p>
      }
    >
      <div className="group inline-block" data-offset-key={props.offsetKey}>
        <span className={`italic font-bold ${clx}`}>{props.children}</span>
      </div>
    </Tippy>
  );
};

export default DraftEditor;
