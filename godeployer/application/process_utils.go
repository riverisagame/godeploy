// ============================================================
// 文件：process_utils.go
// 作用：🛠️ 命令行执行的"工具箱"！
//
// 这个文件只有一个函数 runCmd，但它非常非常重要！
// 它负责在本地执行命令行程序（比如 git clone、npm build 等），
// 并且支持"超时取消"——如果命令执行太久，可以强制杀掉它！
//
// 为什么要单独写一个函数？
// 因为 Go 自带的 exec.Command 在取消时不会自动杀掉"子进程的子进程"。
// 比如你执行 npm build，npm 又会启动很多子进程，
// 如果超时了，这些子进程还会在后台"阴魂不散"……
// 这个函数通过"进程组"机制，确保整个进程树一起被干掉！💀
// ============================================================

// @Ref: docs/sps/plans/20260531-ddd-full-tactical-plan.md | @Date: 2026-05-31

package application

import (
	"bytes"       // 💾 字节缓冲区：用来存命令的输出结果
	"context"     // 📡 上下文：控制超时和取消
	"os/exec"     // 🖥️ 执行外部命令（git、npm、rsync 等）

	"deploy/godeployer/infrastructure/sys" // ⚙️ 系统工具：设置进程组、杀进程
)

// runCmd 本地命令执行包裹，支持 Context 超时下彻底清退子进程树。
//
// 它做了 3 件重要的事：
// 1. 捕获输出：把命令的 stdout 和 stderr 都抓到一个缓冲区里
// 2. 设置进程组：这样以后要杀可以连子子孙孙一起杀
// 3. 监听超时：如果 ctx 被取消（超时/手动取消），果断杀掉整个进程组
//
// @Ref: docs/sps/plans/20260527_nanoplan_resilience.md | @Date: 2026-05-27
func runCmd(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	// 创建一个缓冲区，用来存命令的输出结果
	// 这样命令执行完后我们可以看它输出了什么
	var buf bytes.Buffer
	cmd.Stdout = &buf // 📤 标准输出（正常信息）
	cmd.Stderr = &buf // 📥 标准错误（错误信息）

	// 🏗️ 设置"进程组"：把当前命令和它将来创建的所有子进程
	// 都放进同一个进程组。以后要杀就一起杀！
	sys.SetProcessGroup(cmd)

	// 🚀 启动命令（但不等待它完成）
	// cmd.Start() 启动后，命令在后台运行
	if err := cmd.Start(); err != nil {
		return nil, err // 如果启动失败（比如命令不存在），直接返回错误
	}

	// 创建一个通道（channel），用来接收命令完成的通知
	// done <- cmd.Wait() 的意思是：
	// 等命令执行完，把结果发到 done 这个通道里
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait() // 等待命令完成，然后把错误（可能没有）发到通道
	}()

	// 🔄 select：等两件事中的任意一件发生
	select {
	case <-ctx.Done():
		// 😱 Context 被取消了！（可能是超时，可能是用户取消）
		// 杀掉整个进程组——包括所有子进程
		sys.KillProcessGroup(cmd)
		<-done                          // 等待命令真的被干掉
		return buf.Bytes(), ctx.Err()    // 返回已经输出的内容 + 超时错误

	case err := <-done:
		// ✅ 命令正常执行完了！
		return buf.Bytes(), err // 返回输出内容 + 可能有的错误
	}
}

// ============================================================
// 📚 面试题大全
// ============================================================
// 初级（初中生也能答）：
// 1. Q: 什么是"进程组"（Process Group）？
//    A: 把多个相关进程绑在一起。就像一家人——要搬家就一起搬，
//       要关掉就一起关掉，不会漏掉某个子进程~
//
// 2. Q: channel（通道）是做什么的？
//    A: Go 里面协程（goroutine）之间通信的"水管"。
//       一个协程发消息，另一个协程收消息。done <- err 是发消息~
//
// 中级（面试常考）：
// 3. Q: 为什么 cmd.Stdout 和 cmd.Stderr 都指向同一个缓冲区？
//    A: 保持输出顺序！如果分开两个缓冲区，stdout 和 stderr 的打印顺序会乱。
//       合在一起虽然分不清是正常输出还是错误，但能保证顺序正确~
//
// 4. Q: go func() { done <- cmd.Wait() } 为什么要放在协程里？
//    A: cmd.Wait() 会阻塞等待命令完成。如果不用协程，
//       select 就无法同时监听"超时取消"——会一直卡在 Wait 上。
//       用协程包装后，select 可以同时等待两个事件~
//
// 高级（架构师级别）：
// 5. Q: 为什么需要 KillProcessGroup 而不是直接 cmd.Process.Kill()？
//    A: 杀死父进程通常会让子进程变成"孤儿进程"继续运行！
//       比如 npm build 启动了 webpack 子进程，只杀 npm 没用。
//       进程组机制确保整个进程树一起被清理，不留后患~
// ============================================================
