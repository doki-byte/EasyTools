import React from "react";

const RspText = ({ bodyContent }) => {
  return (
    <div className="pt-2 h-full w-full">
      <div className="h-full w-full p-2 border border-lines overflow-auto">{bodyContent && <pre className="text-txtprim text-sm">{bodyContent}</pre>}</div>
    </div>
  );
};

export default React.memo(RspText);
