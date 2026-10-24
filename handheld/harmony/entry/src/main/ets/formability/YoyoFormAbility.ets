import { formBindingData, FormExtensionAbility, formProvider } from '@kit.FormKit';
import { Want } from '@kit.AbilityKit';

// Thin satellite. The Flutter client owns the workstation port. This card only
// shows the running turn and opens that client.
export default class YoyoFormAbility extends FormExtensionAbility {
  onAddForm(want: Want): formBindingData.FormBindingData {
    return formBindingData.createFormBindingData({
      text: '工作站空闲',
      session: ''
    });
  }

  onUpdateForm(formId: string): void {
    formProvider.updateForm(formId, formBindingData.createFormBindingData({
      text: '回到 Yoyo 查看这一轮',
      session: ''
    })).catch(() => {});
  }

  onFormEvent(formId: string, message: string): void {
    this.context.startAbility({
      bundleName: 'shenchangxin.yoyo.handheld',
      abilityName: 'EntryAbility'
    });
  }
}
