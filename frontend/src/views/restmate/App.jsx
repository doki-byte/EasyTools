import { useEffect } from "react";
import Layout from "./comps/misc/Layout";
import TabsRoot from "./comps/TabsRoot";
import { useStore } from "./store/store";
import "tippy.js/dist/tippy.css";

function App() {
  const loading = useStore((x) => x.appLoading);
  useEffect(() => {
    useStore.getState().getCollections();
    useStore.getState().getEnvs();
  }, []);

  if (loading) {
    return null;
  }
  return (
    <Layout>
      <TabsRoot />
    </Layout>
  );
}

export default App;
