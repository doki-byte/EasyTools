import { hotkeys_json } from "../../utils/utils";

const Shortcuts = () => {
  return (
    <div className="px-6 pb-4 overflow-y-auto">
      <p className="text-txtprim font-bold">Shortcuts</p>
      <div className="w-full mt-4">
        {hotkeys_json.map((c, i) => (
          <div key={i} className="mb-4">
            <div className="flex justify-between items-center">
              <p className="text-xs text-txtprim">{c.name}</p>
              <div className="flex gap-x-2">
                {c.key.map((k, i) => (
                  <pre key={i} className="text-lit bg-sec text-xs py-1 px-1">
                    {k}
                  </pre>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Shortcuts;
