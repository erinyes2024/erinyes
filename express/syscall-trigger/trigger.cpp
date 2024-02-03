#include <unistd.h>
#include <napi.h>
#include <string>
#include <sys/syscall.h>

using namespace Napi;

Value Trigger(const CallbackInfo& info) {
  if (info.Length() < 1 || !info[0].IsString()) {
    TypeError::New(info.Env(), "String argument expected").ThrowAsJavaScriptException();
    return info.Env().Undefined();
  }
  std::string arg0 = info[0].As<String>().Utf8Value();
  syscall(SYS_write, 1, arg0.c_str(), arg0.size());
  return info.Env().Undefined();
}

Object Init(Env env, Object exports) {
  exports.Set(String::New(env, "trigger"), Function::New(env, Trigger));
  return exports;
}

NODE_API_MODULE(trigger, Init)
