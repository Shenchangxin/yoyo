// Package cut 用 ffmpeg、ffprobe 和可选的 whisper.cpp 做本机剪辑：读素材、挑高光、改剪辑表、混音、调色和字幕。
//
// probe.go 读时长、场景切点和静音区间；score.go 按场景切点和语音占比挑不重叠的高光窗口；edit.go 在剪辑表里挖掉或插回一段时间；
// extract.go 按时间抽片段、拼接并加黑边改画幅；audio.go 混入口播和音乐；grade.go 套预设调色；
// captions.go 编码、解析、烧录字幕，有 whisper.cpp 时还会转写音轨。
//
// 外部命令都经 run.go 用配置的 ffmpeg 和 ffprobe 路径执行，并隐藏控制台窗口。这个包不调用生成模型。
package cut
