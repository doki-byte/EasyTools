import { BsRocketTakeoff } from "react-icons/bs";

const AboutUs = () => {
  return (
    <div className="px-6 pb-4 relative">
      <div className="absolute inset-0 w-full h-full z-0">
        <div className="flex justify-start items-end h-full w-full text-txtsec opacity-60">
          <BsRocketTakeoff
            size="160"
            className="hover:text-accent transition-all  duration-300 ease-in-out transform  hover:translate-x-[20px] hover:translate-y-[-20px]"
          />
        </div>
      </div>
      <div className="relative z-10">
        <p className="text-lit font-bold text-2xl tracking-wider">Restmate</p>
        <p className="text-xs text-txtsec">版本 v1.0.0</p>
        <p className="text-xs text-txtsec mt-2 tracking-wider">
          Restmate是一款速度极快、轻量级的REST API客户端，专为那些需要速度和效率而又不影响系统性能的开发人员而构建。
          轻松简化API测试和集成。
        </p>
        <p className="text-xs text-txtsec mt-2 tracking-wider">在您的支持下，帮助我们保持快速发展！</p>
      </div>
    </div>
  );
};

export default AboutUs;
