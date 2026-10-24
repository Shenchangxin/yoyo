@Entry
@Component
struct YoyoCard {
  @LocalStorageProp('text') text: string = '工作站空闲'
  @LocalStorageProp('session') session: string = ''

  build() {
    Column() {
      Text('Yoyo')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor('#F4F4F0')
      Text(this.text)
        .fontSize(13)
        .fontColor('#8C8B85')
        .margin({ top: 8 })
        .maxLines(2)
        .textOverflow({ overflow: TextOverflow.Ellipsis })
      if (this.session.length > 0) {
        Text(this.session)
          .fontSize(12)
          .fontColor('#5C99D6')
          .margin({ top: 6 })
          .maxLines(1)
      }
    }
    .width('100%')
    .height('100%')
    .alignItems(HorizontalAlign.Start)
    .justifyContent(FlexAlign.Center)
    .padding(12)
    .backgroundColor('#1C1C1A')
  }
}
